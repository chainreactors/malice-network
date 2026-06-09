package rpc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/forwardrpc"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func StartForwardListenerClient(ctx context.Context, cfg *configs.ListenerConfig) error {
	if cfg == nil {
		return fmt.Errorf("listener config is nil")
	}
	forwardCfg := cfg.ForwardConfigOrDefault()
	dialOptions, err := forwardrpc.DialOptions(forwardCfg.ConnectHost)
	if err != nil {
		return err
	}
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		dialCtx,
		forwardCfg.ConnectAddress(),
		dialOptions...,
	)
	if err != nil {
		return fmt.Errorf("connect forward listener %s: %w", forwardCfg.ConnectAddress(), err)
	}
	client := forwardrpc.NewForwardListenerClient(conn)
	stream, err := client.ControlStream(ctx)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open forward listener control stream: %w", err)
	}

	lns := ensureForwardListenerRegistered(cfg)
	core.GoGuarded("forward-listener-control-send:"+cfg.Name, func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case msg, ok := <-lns.Ctrl:
				if !ok {
					return nil
				}
				lns.CtrlJob.Store(msg.Id, nil)
				if msg.Job != nil && msg.Job.Pipeline != nil {
					if err := ensureForwardTaskStream(ctx, client, msg.Job.Pipeline.ListenerId, msg.Job.Pipeline.Name); err != nil {
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
	}, core.LogGuardedError("forward-listener-control-send:"+cfg.Name), func() { _ = conn.Close() })

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
				if err := ensureForwardTaskStream(ctx, client, msg.Job.Pipeline.ListenerId, msg.Job.Pipeline.Name); err != nil {
					return err
				}
			}
		}
	}, core.LogGuardedError("forward-listener-control-recv:"+cfg.Name), func() { _ = conn.Close() })

	core.GoGuarded("forward-listener-config-start:"+cfg.Name, func() error {
		return registerAndStartForwardConfiguredPipelines(ctx, cfg)
	}, core.LogGuardedError("forward-listener-config-start:"+cfg.Name))

	logs.Log.Importantf("server.forward - connected listener=%s address=%s", cfg.Name, forwardCfg.ConnectAddress())
	return nil
}

func ensureForwardListenerRegistered(cfg *configs.ListenerConfig) *core.Listener {
	if existing, err := core.Listeners.Get(cfg.Name); err == nil {
		return existing
	}
	lns := core.NewListener(cfg.Name, cfg.IP)
	core.Listeners.Add(lns)
	return lns
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
