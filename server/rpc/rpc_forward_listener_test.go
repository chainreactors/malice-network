package rpc

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/forwardrpc"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	listenerpkg "github.com/chainreactors/malice-network/server/listener"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

type fakeForwardListenerServer struct {
	forwardrpc.UnimplementedForwardListenerServer
	ctrls chan *clientpb.JobCtrl
	tasks chan *clientpb.SpiteRequest
}

func (s *fakeForwardListenerServer) ControlStream(stream forwardrpc.ForwardListener_ControlStreamServer) error {
	for {
		ctrl, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		s.ctrls <- ctrl
		if err := stream.Send(&clientpb.JobStatus{
			ListenerId: ctrl.GetJob().GetPipeline().GetListenerId(),
			Ctrl:       ctrl.Ctrl,
			CtrlId:     ctrl.Id,
			Job:        ctrl.Job,
			Status:     consts.CtrlStatusSuccess,
		}); err != nil {
			return err
		}
	}
}

func (s *fakeForwardListenerServer) TaskStream(stream forwardrpc.ForwardListener_TaskStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		s.tasks <- req
	}
}

func writeForwardAuthConfig(t testing.TB) (string, string) {
	t.Helper()
	auth, fingerprint, err := certutils.GenerateListenerCert("127.0.0.1", "forward-rpc-test", 0)
	if err != nil {
		t.Fatalf("GenerateListenerCert failed: %v", err)
	}
	data, err := yaml.Marshal(auth)
	if err != nil {
		t.Fatalf("marshal auth failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "listener.auth")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write auth failed: %v", err)
	}
	return path, fingerprint
}

func TestStartForwardListenerClientDeliversCtrlAndReceivesStatus(t *testing.T) {
	initForwardRPCTestDB(t)
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)
	t.Cleanup(resetForwardListenerRuntimes)
	oldBroker := core.EventBroker
	core.EventBroker = core.NewBroker()
	t.Cleanup(func() {
		if core.EventBroker != nil {
			core.EventBroker.Stop()
		}
		core.EventBroker = oldBroker
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake forward listener: %v", err)
	}
	authPath, fingerprint := writeForwardAuthConfig(t)
	serverOptions, err := forwardrpc.ServerOptions(authPath)
	if err != nil {
		t.Fatalf("build forward server options: %v", err)
	}
	grpcServer := grpc.NewServer(serverOptions...)
	fake := &fakeForwardListenerServer{
		ctrls: make(chan *clientpb.JobCtrl, 1),
		tasks: make(chan *clientpb.SpiteRequest, 1),
	}
	forwardrpc.RegisterForwardListenerServer(grpcServer, fake)
	go func() { _ = grpcServer.Serve(ln) }()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = ln.Close()
	})

	addr := ln.Addr().(*net.TCPAddr)
	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      "forward-server-test",
		Auth:      authPath,
		IP:        "127.0.0.1",
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ConnectHost: "127.0.0.1",
			ConnectPort: uint16(addr.Port),
		},
	}
	seedForwardListenerOperator(t, cfg.Name, fingerprint)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := StartForwardListenerClient(ctx, cfg); err != nil {
		t.Fatalf("StartForwardListenerClient failed: %v", err)
	}

	lns, err := core.Listeners.Get(cfg.Name)
	if err != nil {
		t.Fatalf("listener was not registered: %v", err)
	}
	pipeline := &clientpb.Pipeline{
		Name:       "custom-forward-server",
		ListenerId: cfg.Name,
		Enable:     true,
		Type:       consts.CustomPipeline,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: "custom-forward-server", ListenerId: cfg.Name},
		},
	}
	ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlPipelineStart,
		Job:  &clientpb.Job{Name: pipeline.Name, Pipeline: pipeline},
	})
	status := lns.WaitCtrl(ctrlID)
	if status == nil || status.Status != consts.CtrlStatusSuccess {
		t.Fatalf("status = %#v, want success", status)
	}

	select {
	case got := <-fake.ctrls:
		if got.Id != ctrlID || got.Job.GetName() != pipeline.Name {
			t.Fatalf("forwarded ctrl = %#v, want id=%d name=%s", got, ctrlID, pipeline.Name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for forwarded ctrl")
	}

	streamVal, ok := pipelinesCh.Load(core.PipelineRuntimeKey(cfg.Name, pipeline.Name))
	if !ok {
		t.Fatalf("forward task stream was not registered")
	}
	taskReq := &clientpb.SpiteRequest{
		Session: &clientpb.Session{SessionId: "session-forward", ListenerId: cfg.Name, PipelineId: pipeline.Name},
		Task:    &clientpb.Task{TaskId: 7, SessionId: "session-forward"},
	}
	if err := streamVal.(grpc.ServerStream).SendMsg(taskReq); err != nil {
		t.Fatalf("send task through forward stream: %v", err)
	}
	select {
	case got := <-fake.tasks:
		if got.GetTask().GetTaskId() != taskReq.Task.TaskId || got.GetSession().GetSessionId() != taskReq.Session.SessionId {
			t.Fatalf("forwarded task = %#v, want %#v", got, taskReq)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for forwarded task")
	}
}

func TestEnsureForwardTaskStreamDoesNotOpenDuplicateConcurrentStreams(t *testing.T) {
	withIsolatedPipelinesCh(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &blockingForwardTaskClient{
		started: make(chan int32, 2),
		release: make(chan struct{}),
	}
	released := false
	release := func() {
		if !released {
			close(client.release)
			released = true
		}
	}
	defer release()

	firstErr := make(chan error, 1)
	go func() {
		firstErr <- ensureForwardTaskStream(ctx, client, "listener-concurrent", "pipeline-concurrent")
	}()

	select {
	case <-client.started:
	case <-time.After(time.Second):
		t.Fatal("first TaskStream call did not start")
	}

	secondErr := make(chan error, 1)
	go func() {
		secondErr <- ensureForwardTaskStream(ctx, client, "listener-concurrent", "pipeline-concurrent")
	}()

	select {
	case call := <-client.started:
		release()
		<-firstErr
		<-secondErr
		t.Fatalf("opened duplicate TaskStream call %d while the first stream was still starting", call)
	case <-time.After(100 * time.Millisecond):
	}

	release()
	if err := <-firstErr; err != nil {
		t.Fatalf("first ensureForwardTaskStream returned error: %v", err)
	}
	if err := <-secondErr; err != nil {
		t.Fatalf("second ensureForwardTaskStream returned error: %v", err)
	}
	if got := client.calls.Load(); got != 1 {
		t.Fatalf("TaskStream opened %d times, want 1", got)
	}
	cancel()
}

func TestResetForwardListenerRuntimesWaitsForInFlightStop(t *testing.T) {
	resetForwardListenerRuntimes()
	withIsolatedListenersAndJobs(t)

	const listenerID = "forward-stop-reset-race"
	core.Listeners.Add(core.NewListener(listenerID, "127.0.0.1"))
	stopEntered := make(chan struct{})
	releaseStop := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseStop) }) }
	t.Cleanup(release)

	runtime := &forwardListenerRuntime{
		listenerID:   listenerID,
		ownsListener: true,
		cancel: func() {
			close(stopEntered)
			<-releaseStop
		},
	}
	forwardListenerRuntimes.Store(listenerID, runtime)

	stopDone := make(chan struct{})
	go func() {
		runtime.stop()
		close(stopDone)
	}()
	select {
	case <-stopEntered:
	case <-time.After(time.Second):
		t.Fatal("runtime stop did not reach the deterministic barrier")
	}

	resetDone := make(chan struct{})
	go func() {
		resetForwardListenerRuntimes()
		close(resetDone)
	}()
	select {
	case <-resetDone:
		t.Fatal("reset returned while the runtime stop was still in flight")
	case <-time.After(25 * time.Millisecond):
	}

	release()
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("runtime stop did not finish after releasing the barrier")
	}
	select {
	case <-resetDone:
	case <-time.After(time.Second):
		t.Fatal("reset did not finish after runtime stop completed")
	}
}

type blockingForwardTaskClient struct {
	calls   atomic.Int32
	started chan int32
	release chan struct{}
}

func (c *blockingForwardTaskClient) ControlStream(context.Context, ...grpc.CallOption) (forwardrpc.ForwardListener_ControlStreamClient, error) {
	return nil, errors.New("not used")
}

func (c *blockingForwardTaskClient) TaskStream(ctx context.Context, _ ...grpc.CallOption) (forwardrpc.ForwardListener_TaskStreamClient, error) {
	call := c.calls.Add(1)
	c.started <- call
	select {
	case <-c.release:
		return &blockingForwardTaskStream{ctx: ctx}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type blockingForwardTaskStream struct {
	grpc.ClientStream
	ctx context.Context
}

func (s *blockingForwardTaskStream) Send(*clientpb.SpiteRequest) error {
	return nil
}

func (s *blockingForwardTaskStream) Recv() (*clientpb.SpiteRequest, error) {
	<-s.ctx.Done()
	return nil, s.ctx.Err()
}

type reconnectingForwardTaskClient struct {
	streams []*reconnectingForwardTaskStream
	calls   atomic.Int32
}

func (c *reconnectingForwardTaskClient) ControlStream(context.Context, ...grpc.CallOption) (forwardrpc.ForwardListener_ControlStreamClient, error) {
	return nil, errors.New("not used")
}

func (c *reconnectingForwardTaskClient) TaskStream(ctx context.Context, _ ...grpc.CallOption) (forwardrpc.ForwardListener_TaskStreamClient, error) {
	index := int(c.calls.Add(1)) - 1
	if index >= len(c.streams) {
		return nil, errors.New("no more task streams")
	}
	stream := c.streams[index]
	stream.ctx = ctx
	return stream, nil
}

type reconnectingForwardTaskStream struct {
	grpc.ClientStream
	ctx     context.Context
	recvErr error
	entered chan struct{}
	sent    chan *clientpb.SpiteRequest
	once    sync.Once
}

func (s *reconnectingForwardTaskStream) Send(req *clientpb.SpiteRequest) error {
	s.sent <- req
	return nil
}

func (s *reconnectingForwardTaskStream) Recv() (*clientpb.SpiteRequest, error) {
	s.once.Do(func() { close(s.entered) })
	if s.recvErr != nil {
		return nil, s.recvErr
	}
	<-s.ctx.Done()
	return nil, s.ctx.Err()
}

func TestForwardTaskStreamReconnectsAndKeepsReplacementRegistered(t *testing.T) {
	withIsolatedPipelinesCh(t)

	first := &reconnectingForwardTaskStream{
		recvErr: errors.New("connection reset"),
		entered: make(chan struct{}),
		sent:    make(chan *clientpb.SpiteRequest, 1),
	}
	second := &reconnectingForwardTaskStream{
		entered: make(chan struct{}),
		sent:    make(chan *clientpb.SpiteRequest, 1),
	}
	client := &reconnectingForwardTaskClient{streams: []*reconnectingForwardTaskStream{first, second}}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const listenerID = "listener-task-reconnect"
	const pipelineID = "pipeline-task-reconnect"
	if err := ensureForwardTaskStream(ctx, client, listenerID, pipelineID); err != nil {
		t.Fatalf("ensureForwardTaskStream failed: %v", err)
	}

	select {
	case <-second.entered:
	case <-time.After(2 * time.Second):
		t.Fatalf("replacement task stream did not start, open count = %d", client.calls.Load())
	}

	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	streamValue, ok := pipelinesCh.Load(key)
	if !ok {
		t.Fatal("replacement task stream is not registered")
	}
	adapter, ok := streamValue.(*forwardTaskServerStream)
	if !ok || adapter.stream != second {
		t.Fatalf("registered stream = %#v, want second generation", streamValue)
	}

	want := &clientpb.SpiteRequest{Task: &clientpb.Task{TaskId: 41}}
	if err := adapter.SendMsg(want); err != nil {
		t.Fatalf("send through replacement stream failed: %v", err)
	}
	select {
	case got := <-second.sent:
		if got.GetTask().GetTaskId() != want.GetTask().GetTaskId() {
			t.Fatalf("replacement stream task id = %d, want %d", got.GetTask().GetTaskId(), want.GetTask().GetTaskId())
		}
	case <-time.After(time.Second):
		t.Fatal("replacement task stream did not receive task")
	}

	cancel()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, exists := pipelinesCh.Load(key); !exists {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("replacement task stream remained registered after parent context cancellation")
}

func TestRetireListenerThroughForwardControlStream(t *testing.T) {
	initForwardRPCTestDB(t)
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)
	t.Cleanup(resetForwardListenerRuntimes)
	oldBroker := core.EventBroker
	core.EventBroker = core.NewBroker()
	t.Cleanup(func() {
		if core.EventBroker != nil {
			core.EventBroker.Stop()
		}
		core.EventBroker = oldBroker
	})

	portListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve forward listener port: %v", err)
	}
	port := uint16(portListener.Addr().(*net.TCPAddr).Port)
	_ = portListener.Close()

	authPath, fingerprint := writeForwardAuthConfig(t)
	configPath := filepath.Join(t.TempDir(), "listener.yaml")
	if err := os.WriteFile(configPath, []byte("listeners: {}\n"), 0600); err != nil {
		t.Fatalf("write listener config: %v", err)
	}
	oldConfigFilename := configs.CurrentServerConfigFilename
	configs.CurrentServerConfigFilename = configPath
	t.Cleanup(func() { configs.CurrentServerConfigFilename = oldConfigFilename })

	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      "forward-retire-e2e",
		Auth:      authPath,
		IP:        "127.0.0.1",
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ListenHost:  "127.0.0.1",
			ListenPort:  port,
			ConnectHost: "127.0.0.1",
			ConnectPort: port,
		},
	}
	forwardRuntime, err := listenerpkg.NewForwardListener(cfg)
	if err != nil {
		t.Fatalf("NewForwardListener failed: %v", err)
	}
	t.Cleanup(func() { _ = forwardRuntime.Close() })

	seedForwardAdminOperator(t, "admin-retire-e2e", "admin-retire-e2e-fp")
	seedForwardListenerOperator(t, cfg.Name, fingerprint)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := StartForwardListenerClient(ctx, cfg); err != nil {
		t.Fatalf("StartForwardListenerClient failed: %v", err)
	}

	adminCtx := contextWithIdentity(context.Background(), &PeerIdentity{Fingerprint: "admin-retire-e2e-fp"})
	reply, err := (&Server{}).RetireListener(adminCtx, &clientpb.ListenerRetire{
		ListenerId:     cfg.Name,
		PurgeConfig:    true,
		PurgeAuth:      true,
		TimeoutSeconds: 3,
	})
	if err != nil {
		t.Fatalf("RetireListener failed: %v", err)
	}
	if reply.GetListenerId() != cfg.Name || reply.GetActive() {
		t.Fatalf("reply = %#v, want inactive %s", reply, cfg.Name)
	}
	if _, ok := getForwardListenerRuntime(cfg.Name); ok {
		t.Fatalf("forward runtime for %s still registered", cfg.Name)
	}
	if _, err := core.Listeners.Get(cfg.Name); err == nil {
		t.Fatalf("core listener %s still registered", cfg.Name)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("config stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(authPath); !os.IsNotExist(err) {
		t.Fatalf("auth stat error = %v, want not exist", err)
	}
	operator, err := db.FindOperatorByName(cfg.Name)
	if err != nil {
		t.Fatalf("FindOperatorByName failed: %v", err)
	}
	if !operator.Revoked {
		t.Fatalf("listener operator %s was not revoked", cfg.Name)
	}
}

func TestStartForwardListenerClientRejectsUnexpectedListenerFingerprint(t *testing.T) {
	initForwardRPCTestDB(t)
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)
	t.Cleanup(resetForwardListenerRuntimes)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake forward listener: %v", err)
	}
	authPath, _ := writeForwardAuthConfig(t)
	serverOptions, err := forwardrpc.ServerOptions(authPath)
	if err != nil {
		t.Fatalf("build forward server options: %v", err)
	}
	grpcServer := grpc.NewServer(serverOptions...)
	forwardrpc.RegisterForwardListenerServer(grpcServer, &fakeForwardListenerServer{
		ctrls: make(chan *clientpb.JobCtrl, 1),
		tasks: make(chan *clientpb.SpiteRequest, 1),
	})
	go func() { _ = grpcServer.Serve(ln) }()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = ln.Close()
	})

	addr := ln.Addr().(*net.TCPAddr)
	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      "forward-fp-mismatch",
		Auth:      authPath,
		IP:        "127.0.0.1",
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ConnectHost: "127.0.0.1",
			ConnectPort: uint16(addr.Port),
		},
	}
	seedForwardListenerOperator(t, cfg.Name, "deadbeef")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = StartForwardListenerClient(ctx, cfg)
	if err == nil {
		t.Fatal("StartForwardListenerClient succeeded with mismatched listener fingerprint")
	}
}

func TestRequireAdminRoleRejectsOperatorRole(t *testing.T) {
	initForwardRPCTestDB(t)
	if err := db.CreateOperator(&models.Operator{
		Name:        "plain-operator",
		Type:        mtls.Client,
		Role:        models.RoleOperator,
		Fingerprint: "operator-forward-fp",
	}); err != nil {
		t.Fatalf("CreateOperator failed: %v", err)
	}
	opCache.Invalidate()
	ctx := contextWithIdentity(context.Background(), &PeerIdentity{Fingerprint: "operator-forward-fp"})

	err := requireAdminRole(ctx)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("requireAdminRole error = %v, want PermissionDenied", err)
	}
}

func TestConnectForwardListenerRequiresAdminAndRegisteredListener(t *testing.T) {
	initForwardRPCTestDB(t)
	seedForwardAdminOperator(t, "admin-client", "admin-forward-fp")

	_, err := (&Server{}).ConnectForwardListener(
		contextWithIdentity(context.Background(), &PeerIdentity{Fingerprint: "admin-forward-fp"}),
		&clientpb.ForwardListenerConnect{
			ListenerId:  "missing-listener",
			ConnectHost: "127.0.0.1",
			ConnectPort: 5005,
		},
	)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("ConnectForwardListener error = %v, want NotFound for missing listener", err)
	}
}

func TestConnectForwardListenerRejectsMissingHostAndPortOverflow(t *testing.T) {
	initForwardRPCTestDB(t)
	seedForwardAdminOperator(t, "admin-client", "admin-forward-fp")
	seedForwardListenerOperator(t, "forward-input-listener", "listener-forward-fp")
	ctx := contextWithIdentity(context.Background(), &PeerIdentity{Fingerprint: "admin-forward-fp"})

	_, err := (&Server{}).ConnectForwardListener(ctx, &clientpb.ForwardListenerConnect{
		ListenerId:     "forward-input-listener",
		TimeoutSeconds: 1,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ConnectForwardListener missing host error = %v, want InvalidArgument", err)
	}

	_, err = (&Server{}).ConnectForwardListener(ctx, &clientpb.ForwardListenerConnect{
		ListenerId:     "forward-input-listener",
		ConnectHost:    "127.0.0.1",
		ConnectPort:    70000,
		TimeoutSeconds: 1,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ConnectForwardListener port overflow error = %v, want InvalidArgument", err)
	}
}

func TestStartForwardListenerClientRejectsActiveCoreListenerCollision(t *testing.T) {
	initForwardRPCTestDB(t)
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)
	t.Cleanup(resetForwardListenerRuntimes)
	oldBroker := core.EventBroker
	core.EventBroker = core.NewBroker()
	t.Cleanup(func() {
		if core.EventBroker != nil {
			core.EventBroker.Stop()
		}
		core.EventBroker = oldBroker
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake forward listener: %v", err)
	}
	authPath, fingerprint := writeForwardAuthConfig(t)
	serverOptions, err := forwardrpc.ServerOptions(authPath)
	if err != nil {
		t.Fatalf("build forward server options: %v", err)
	}
	grpcServer := grpc.NewServer(serverOptions...)
	forwardrpc.RegisterForwardListenerServer(grpcServer, &fakeForwardListenerServer{
		ctrls: make(chan *clientpb.JobCtrl, 1),
		tasks: make(chan *clientpb.SpiteRequest, 1),
	})
	go func() { _ = grpcServer.Serve(ln) }()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = ln.Close()
	})

	addr := ln.Addr().(*net.TCPAddr)
	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      "listener-collision",
		Auth:      authPath,
		IP:        "127.0.0.1",
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ConnectHost: "127.0.0.1",
			ConnectPort: uint16(addr.Port),
		},
	}
	seedForwardListenerOperator(t, cfg.Name, fingerprint)
	core.Listeners.Add(core.NewListener(cfg.Name, "10.0.0.9"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = StartForwardListenerClient(ctx, cfg)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("StartForwardListenerClient collision error = %v, want FailedPrecondition", err)
	}
}

func TestStaleForwardRuntimeStopKeepsReplacementState(t *testing.T) {
	newRPCTestEnv(t)
	withIsolatedPipelinesCh(t)
	t.Cleanup(resetForwardListenerRuntimes)

	const listenerID = "forward-runtime-replacement"
	replacementListener := core.NewListener(listenerID, "10.0.0.9")
	core.Listeners.Add(replacementListener)
	replacementRuntime := &forwardListenerRuntime{listenerID: listenerID, ownsListener: true}
	forwardListenerRuntimes.Store(listenerID, replacementRuntime)
	pipelineKey := core.PipelineRuntimeKey(listenerID, "replacement-pipeline")
	replacementStream := &testRPCServerStream{}
	pipelinesCh.Store(pipelineKey, replacementStream)

	staleRuntime := &forwardListenerRuntime{listenerID: listenerID, ownsListener: true}
	staleRuntime.stop()

	if got, ok := forwardListenerRuntimes.Load(listenerID); !ok || got != replacementRuntime {
		t.Fatalf("replacement runtime = %#v, present=%v; stale stop removed it", got, ok)
	}
	if got, err := core.Listeners.Get(listenerID); err != nil || got != replacementListener || !got.Active() {
		t.Fatalf("replacement listener = %#v, error=%v; stale stop changed it", got, err)
	}
	if got, ok := pipelinesCh.Load(pipelineKey); !ok || got != replacementStream {
		t.Fatalf("replacement pipeline stream = %#v, present=%v; stale stop removed it", got, ok)
	}
}

func seedForwardAdminOperator(t testing.TB, name, fingerprint string) {
	t.Helper()
	if err := db.CreateOperator(&models.Operator{
		Name:        name,
		Type:        mtls.Client,
		Role:        models.RoleAdmin,
		Fingerprint: fingerprint,
	}); err != nil {
		t.Fatalf("CreateOperator admin failed: %v", err)
	}
	opCache.Invalidate()
}

func seedForwardListenerOperator(t testing.TB, listenerID, fingerprint string) {
	t.Helper()
	if err := db.CreateOperator(&models.Operator{
		Name:        listenerID,
		Remote:      "127.0.0.1",
		Type:        mtls.Listener,
		Role:        models.RoleListener,
		Fingerprint: fingerprint,
	}); err != nil {
		t.Fatalf("CreateOperator listener failed: %v", err)
	}
	opCache.Invalidate()
}

func initForwardRPCTestDB(t testing.TB) {
	t.Helper()
	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	if err := os.MkdirAll(configs.ServerRootPath, 0700); err != nil {
		t.Fatalf("create test root failed: %v", err)
	}
	client, err := db.NewDBClient(nil)
	if err != nil {
		t.Fatalf("NewDBClient failed: %v", err)
	}
	db.Client = client
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	opCache.Invalidate()
}
