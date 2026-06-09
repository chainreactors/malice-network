package rpc

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
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
