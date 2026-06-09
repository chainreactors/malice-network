package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/forwardrpc"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	errForwardListenerAlreadyConnected = errors.New("forward listener already connected")
	forwardListenerRuntimeMu           sync.Mutex
	forwardListenerRuntimes            sync.Map
)

type forwardListenerRuntime struct {
	listenerID   string
	connectHost  string
	connectPort  uint32
	address      string
	fingerprint  string
	cancel       context.CancelFunc
	conn         *grpc.ClientConn
	ownsListener bool
	stopOnce     sync.Once
}

func StartForwardListenerClient(ctx context.Context, cfg *configs.ListenerConfig) error {
	_, err := startForwardListenerClient(ctx, cfg, 5*time.Second)
	return err
}

func startForwardListenerClient(ctx context.Context, cfg *configs.ListenerConfig, timeout time.Duration) (*forwardListenerRuntime, error) {
	if cfg == nil {
		return nil, fmt.Errorf("listener config is nil")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return nil, fmt.Errorf("listener name is empty")
	}
	listenerOp, err := loadForwardListenerOperator(cfg.Name)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if _, ok := forwardListenerRuntimes.Load(cfg.Name); ok {
		return nil, errForwardListenerAlreadyConnected
	}
	forwardCfg := cfg.ForwardConfigOrDefault()
	dialOptions, err := forwardrpc.DialOptions(forwardCfg.ConnectHost, listenerOp.Fingerprint)
	if err != nil {
		return nil, err
	}
	dialCtx, cancelDial := context.WithTimeout(ctx, timeout)
	defer cancelDial()
	conn, err := grpc.DialContext(
		dialCtx,
		forwardCfg.ConnectAddress(),
		dialOptions...,
	)
	if err != nil {
		return nil, fmt.Errorf("connect forward listener %s: %w", forwardCfg.ConnectAddress(), err)
	}
	runtimeCtx, cancelRuntime := context.WithCancel(context.Background())
	client := forwardrpc.NewForwardListenerClient(conn)
	stream, err := client.ControlStream(runtimeCtx)
	if err != nil {
		cancelRuntime()
		_ = conn.Close()
		return nil, fmt.Errorf("open forward listener control stream: %w", err)
	}
	runtime := &forwardListenerRuntime{
		listenerID:  cfg.Name,
		connectHost: forwardCfg.ConnectHost,
		connectPort: uint32(forwardCfg.ConnectPort),
		address:     forwardCfg.ConnectAddress(),
		fingerprint: listenerOp.Fingerprint,
		cancel:      cancelRuntime,
		conn:        conn,
	}
	forwardListenerRuntimeMu.Lock()
	if _, ok := forwardListenerRuntimes.Load(cfg.Name); ok {
		forwardListenerRuntimeMu.Unlock()
		runtime.stop()
		return nil, errForwardListenerAlreadyConnected
	}
	lns, ownsListener, err := ensureForwardListenerRegistered(cfg)
	if err != nil {
		forwardListenerRuntimeMu.Unlock()
		runtime.stop()
		return nil, err
	}
	runtime.ownsListener = ownsListener
	forwardListenerRuntimes.Store(cfg.Name, runtime)
	forwardListenerRuntimeMu.Unlock()

	core.GoGuarded("forward-listener-control-send:"+cfg.Name, func() error {
		for {
			select {
			case <-runtimeCtx.Done():
				return nil
			case msg, ok := <-lns.Ctrl:
				if !ok {
					return nil
				}
				lns.CtrlJob.Store(msg.Id, nil)
				if msg.Job != nil && msg.Job.Pipeline != nil {
					if err := ensureForwardTaskStream(runtimeCtx, client, msg.Job.Pipeline.ListenerId, msg.Job.Pipeline.Name); err != nil {
						lns.CtrlJob.Delete(msg.Id)
						return err
					}
				}
				if err := stream.Send(msg); err != nil {
					lns.CtrlJob.Delete(msg.Id)
					return fmt.Errorf("send forward listener ctrl: %w", err)
				}
			}
		}
	}, core.LogGuardedError("forward-listener-control-send:"+cfg.Name), runtime.stop)

	core.GoGuarded("forward-listener-control-recv:"+cfg.Name, func() error {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("recv forward listener status: %w", err)
			}
			handleJobStatus(lns, msg)
			if msg.Status == consts.CtrlStatusSuccess && msg.Job != nil && msg.Job.Pipeline != nil {
				if err := ensureForwardTaskStream(runtimeCtx, client, msg.Job.Pipeline.ListenerId, msg.Job.Pipeline.Name); err != nil {
					return err
				}
			}
		}
	}, core.LogGuardedError("forward-listener-control-recv:"+cfg.Name), runtime.stop)

	core.GoGuarded("forward-listener-config-start:"+cfg.Name, func() error {
		return registerAndStartForwardConfiguredPipelines(runtimeCtx, cfg)
	}, core.LogGuardedError("forward-listener-config-start:"+cfg.Name))

	logs.Log.Importantf("server.forward - connected listener=%s address=%s", cfg.Name, forwardCfg.ConnectAddress())
	return runtime, nil
}

func ensureForwardListenerRegistered(cfg *configs.ListenerConfig) (*core.Listener, bool, error) {
	if existing, err := core.Listeners.Get(cfg.Name); err == nil {
		if existing.Active() {
			return nil, false, status.Errorf(codes.FailedPrecondition, "listener %s is already active", cfg.Name)
		}
		for _, pipe := range existing.AllPipelines() {
			deletePipelineStream(pipe.ListenerId, pipe.Name)
		}
		_ = core.Listeners.Stop(cfg.Name)
		core.Listeners.Map.Delete(cfg.Name)
	}
	lns := core.NewListener(cfg.Name, cfg.IP)
	core.Listeners.Add(lns)
	return lns, true, nil
}

func loadForwardListenerOperator(listenerID string) (*models.Operator, error) {
	listenerID = strings.TrimSpace(listenerID)
	if listenerID == "" {
		return nil, status.Error(codes.InvalidArgument, "listener_id is required")
	}
	op, err := db.FindOperatorByName(listenerID)
	if err != nil || op == nil {
		return nil, status.Errorf(codes.NotFound, "listener %s is not registered", listenerID)
	}
	if op.Type != mtls.Listener || op.Role != models.RoleListener {
		return nil, status.Errorf(codes.FailedPrecondition, "operator %s is not a listener identity", listenerID)
	}
	if op.Revoked {
		return nil, status.Errorf(codes.FailedPrecondition, "listener %s has been revoked", listenerID)
	}
	if strings.TrimSpace(op.Fingerprint) == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "listener %s is missing certificate fingerprint", listenerID)
	}
	return op, nil
}

func (r *forwardListenerRuntime) stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		forwardListenerRuntimes.Delete(r.listenerID)
		clearForwardTaskStreams(r.listenerID)
		if r.cancel != nil {
			r.cancel()
		}
		if r.conn != nil {
			_ = r.conn.Close()
		}
		if r.ownsListener {
			listener, err := core.Listeners.Get(r.listenerID)
			if err != nil {
				return
			}
			_ = core.Listeners.Stop(r.listenerID)
			core.Listeners.Remove(listener)
		}
	})
}

func stopForwardListenerClient(listenerID string) (*clientpb.ForwardListenerStatus, error) {
	listenerID = strings.TrimSpace(listenerID)
	if listenerID == "" {
		return nil, status.Error(codes.InvalidArgument, "listener_id is required")
	}
	val, ok := forwardListenerRuntimes.Load(listenerID)
	if !ok {
		return inactiveForwardListenerStatus(listenerID), status.Errorf(codes.NotFound, "forward listener %s is not connected", listenerID)
	}
	runtime := val.(*forwardListenerRuntime)
	status := runtime.toProto()
	status.Active = false
	runtime.stop()
	return status, nil
}

func resetForwardListenerRuntimes() {
	forwardListenerRuntimeMu.Lock()
	defer forwardListenerRuntimeMu.Unlock()
	forwardListenerRuntimes.Range(func(_, value interface{}) bool {
		value.(*forwardListenerRuntime).stop()
		return true
	})
	forwardListenerRuntimes = sync.Map{}
}

func getForwardListenerRuntime(listenerID string) (*forwardListenerRuntime, bool) {
	val, ok := forwardListenerRuntimes.Load(strings.TrimSpace(listenerID))
	if !ok {
		return nil, false
	}
	return val.(*forwardListenerRuntime), true
}

func clearForwardTaskStreams(listenerID string) {
	prefix := listenerID + ":"
	pipelinesCh.Range(func(key, _ interface{}) bool {
		if k, ok := key.(string); ok && strings.HasPrefix(k, prefix) {
			pipelinesCh.Delete(k)
		}
		return true
	})
}

func (r *forwardListenerRuntime) toProto() *clientpb.ForwardListenerStatus {
	return &clientpb.ForwardListenerStatus{
		ListenerId:  r.listenerID,
		ConnectHost: r.connectHost,
		ConnectPort: r.connectPort,
		Address:     r.address,
		Active:      true,
		Fingerprint: r.fingerprint,
	}
}

func inactiveForwardListenerStatus(listenerID string) *clientpb.ForwardListenerStatus {
	return &clientpb.ForwardListenerStatus{
		ListenerId: listenerID,
		Active:     false,
	}
}

func registerAndStartForwardConfiguredPipelines(ctx context.Context, cfg *configs.ListenerConfig) error {
	server := &Server{}
	for _, tcpCfg := range cfg.TcpPipelines {
		pipeline, err := tcpCfg.ToProtobuf(cfg.Name)
		if err != nil {
			return err
		}
		if err := registerAndMaybeStartForwardPipeline(ctx, server, pipeline); err != nil {
			return err
		}
	}
	for _, httpCfg := range cfg.HttpPipelines {
		pipeline, err := httpCfg.ToProtobuf(cfg.Name)
		if err != nil {
			return err
		}
		if err := registerAndMaybeStartForwardPipeline(ctx, server, pipeline); err != nil {
			return err
		}
	}
	for _, bindCfg := range cfg.BindPipelineConfig {
		pipeline, err := bindCfg.ToProtobuf(cfg.Name)
		if err != nil {
			return err
		}
		if err := registerAndMaybeStartForwardPipeline(ctx, server, pipeline); err != nil {
			return err
		}
	}
	return nil
}

func registerAndMaybeStartForwardPipeline(ctx context.Context, server *Server, pipeline *clientpb.Pipeline) error {
	if pipeline == nil {
		return nil
	}
	if _, err := server.RegisterPipeline(ctx, pipeline); err != nil {
		return err
	}
	if pipeline.Tls != nil && pipeline.Tls.Enable && !pipeline.Tls.Acme {
		if _, err := server.GenerateSelfCert(ctx, pipeline); err != nil {
			return err
		}
	} else if pipeline.Tls != nil && pipeline.Tls.Enable && pipeline.Tls.Acme {
		if _, err := server.GenerateAcmeCert(ctx, pipeline); err != nil {
			return err
		}
	}
	if !pipeline.Enable {
		return nil
	}
	_, err := server.StartPipeline(ctx, &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
		Pipeline:   pipeline,
	})
	return err
}

func ensureForwardTaskStream(ctx context.Context, client forwardrpc.ForwardListenerClient, listenerID, pipelineID string) error {
	if listenerID == "" || pipelineID == "" {
		return nil
	}
	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	if _, ok := pipelinesCh.Load(key); ok {
		return nil
	}
	streamCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"listener_id", listenerID,
		"pipeline_id", pipelineID,
	))
	stream, err := client.TaskStream(streamCtx)
	if err != nil {
		return fmt.Errorf("open forward listener task stream %s: %w", key, err)
	}
	adapter := &forwardTaskServerStream{
		ctx:    streamCtx,
		stream: stream,
		key:    key,
	}
	pipelinesCh.Store(key, adapter)
	core.GoGuarded("forward-listener-task-recv:"+key, adapter.receiveLoop, core.LogGuardedError("forward-listener-task-recv:"+key), func() {
		pipelinesCh.Delete(key)
	})
	return nil
}

type forwardTaskServerStream struct {
	ctx    context.Context
	stream forwardrpc.ForwardListener_TaskStreamClient
	key    string
	sendMu sync.Mutex
}

func (s *forwardTaskServerStream) SetHeader(metadata.MD) error  { return nil }
func (s *forwardTaskServerStream) SendHeader(metadata.MD) error { return nil }
func (s *forwardTaskServerStream) SetTrailer(metadata.MD)       {}
func (s *forwardTaskServerStream) Context() context.Context     { return s.ctx }

func (s *forwardTaskServerStream) SendMsg(m interface{}) error {
	req, ok := m.(*clientpb.SpiteRequest)
	if !ok {
		return fmt.Errorf("forward task stream expects *clientpb.SpiteRequest, got %T", m)
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.stream.Send(req)
}

func (s *forwardTaskServerStream) RecvMsg(interface{}) error {
	return fmt.Errorf("forward task stream RecvMsg is not supported")
}

func (s *forwardTaskServerStream) receiveLoop() error {
	for {
		event, err := s.stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := handleForwardTaskEvent(event); err != nil {
			logs.Log.Warnf("forward listener event %s failed: %v", s.key, err)
		}
	}
}

func handleForwardTaskEvent(event *clientpb.SpiteRequest) error {
	if event == nil || event.Spite == nil || event.Session == nil {
		return nil
	}
	session := event.Session
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"session_id", session.SessionId,
		"listener_id", session.ListenerId,
		"pipeline_id", session.PipelineId,
		"timestamp", fmt.Sprintf("%d", time.Now().Unix()),
	))
	switch body := event.Spite.Body.(type) {
	case *implantpb.Spite_Register:
		_, err := (&Server{}).Register(ctx, &clientpb.RegisterSession{
			SessionId:    session.SessionId,
			PipelineId:   session.PipelineId,
			ListenerId:   session.ListenerId,
			Target:       session.Target,
			RawId:        session.RawId,
			Type:         session.Type,
			RegisterData: body.Register,
		})
		return err
	case *implantpb.Spite_Ping:
		_, err := (&Server{}).Checkin(ctx, body.Ping)
		return err
	default:
		return deliverForwardTaskResponse(event)
	}
}

func deliverForwardTaskResponse(event *clientpb.SpiteRequest) error {
	sess, err := core.Sessions.Get(event.Session.SessionId)
	if err != nil {
		dbSess, dbErr := db.FindSession(event.Session.SessionId)
		if dbErr != nil || dbSess == nil {
			return fmt.Errorf("session %s not found in memory or DB", event.Session.SessionId)
		}
		sess, err = core.RecoverSession(dbSess)
		if err != nil {
			return err
		}
		core.Sessions.Add(sess)
	}
	sess.SetLastCheckin(time.Now().Unix())
	if sess.MarkAlive() {
		if err := sess.Save(); err != nil {
			logs.Log.Errorf("save session %s reborn state failed: %s", sess.ID, err.Error())
		}
		sess.Publish(consts.CtrlSessionReborn, fmt.Sprintf("session %s from %s reborn at %s", sess.Abstract(), sess.Target, sess.PipelineID), true, true)
	}
	taskID := event.Spite.TaskId
	if taskID == 0 && event.Task != nil {
		taskID = event.Task.TaskId
	}
	ch, ok := sess.GetResp(taskID)
	if !ok {
		return fmt.Errorf("response channel missing for session %s task %d", event.Session.SessionId, taskID)
	}
	return deliverSpiteResponse(ch, proto.Clone(event.Spite).(*implantpb.Spite))
}

var _ grpc.ServerStream = (*forwardTaskServerStream)(nil)
