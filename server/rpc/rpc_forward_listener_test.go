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
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/forwardrpc"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
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

func writeForwardAuthConfig(t testing.TB) string {
	t.Helper()
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), "malice"))
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	auth, _, err := certutils.GenerateListenerCert("127.0.0.1", "forward-rpc-test", 0)
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
	return path
}

func TestStartForwardListenerClientDeliversCtrlAndReceivesStatus(t *testing.T) {
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)
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
	authPath := writeForwardAuthConfig(t)
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
