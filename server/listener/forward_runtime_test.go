package listener

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/server/forwardrpc"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

func mustOpenForwardLocalStream(t testing.TB, registry *forwardStreamRegistry, listenerID, pipelineID string) *forwardLocalStream {
	t.Helper()
	stream, err := registry.open(listenerID, pipelineID)
	if err != nil {
		t.Fatalf("open forward stream failed: %v", err)
	}
	return stream
}

func reserveForwardPort(t testing.TB) uint16 {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer ln.Close()
	return uint16(ln.Addr().(*net.TCPAddr).Port)
}

func writeForwardAuthConfig(t testing.TB) string {
	t.Helper()
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), "malice"))
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	auth, _, err := certutils.GenerateListenerCert("127.0.0.1", "forward-listener-test", 0)
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

func writeClientOnlyForwardAuthConfig(t testing.TB) string {
	t.Helper()
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), "malice"))
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	ca, caKey, err := certutils.GetCertificateAuthority()
	if err != nil {
		t.Fatalf("GetCertificateAuthority failed: %v", err)
	}
	auth, _, err := certutils.GenerateListenerCert("127.0.0.1", "forward-listener-test", 0)
	if err != nil {
		t.Fatalf("GenerateListenerCert failed: %v", err)
	}
	certPEM, keyPEM, err := certs.GenerateChildCert("127.0.0.1", true, ca, caKey)
	if err != nil {
		t.Fatalf("GenerateChildCert failed: %v", err)
	}
	auth.Certificate = string(certPEM)
	auth.PrivateKey = string(keyPEM)
	data, err := yaml.Marshal(auth)
	if err != nil {
		t.Fatalf("marshal auth failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "listener-client-only.auth")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write auth failed: %v", err)
	}
	return path
}

func TestForwardListenerControlStreamStartsCustomPipeline(t *testing.T) {
	port := reserveForwardPort(t)
	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      "forward-listener-test",
		Auth:      writeForwardAuthConfig(t),
		IP:        "127.0.0.1",
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ListenHost:  "127.0.0.1",
			ListenPort:  port,
			ConnectHost: "127.0.0.1",
			ConnectPort: port,
		},
	}
	runtime, err := NewForwardListener(cfg)
	if err != nil {
		t.Fatalf("NewForwardListener failed: %v", err)
	}
	t.Cleanup(func() { _ = runtime.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialOptions, err := forwardrpc.DialOptions(cfg.ForwardConfigOrDefault().ConnectHost)
	if err != nil {
		t.Fatalf("build forward dial options: %v", err)
	}
	conn, err := grpc.DialContext(ctx, cfg.ForwardConfigOrDefault().ConnectAddress(),
		dialOptions...,
	)
	if err != nil {
		t.Fatalf("dial forward listener: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	stream, err := forwardrpc.NewForwardListenerClient(conn).ControlStream(ctx)
	if err != nil {
		t.Fatalf("open control stream: %v", err)
	}

	pipeline := &clientpb.Pipeline{
		Name:       "custom-forward",
		ListenerId: cfg.Name,
		Enable:     true,
		Type:       consts.CustomPipeline,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: "custom-forward", ListenerId: cfg.Name},
		},
		Tls:        &clientpb.TLS{},
		Encryption: []*clientpb.Encryption{},
		Secure:     &clientpb.Secure{},
	}
	if err := stream.Send(&clientpb.JobCtrl{
		Id:   1,
		Ctrl: consts.CtrlPipelineStart,
		Job:  &clientpb.Job{Name: pipeline.Name, Pipeline: pipeline},
	}); err != nil {
		t.Fatalf("send start ctrl: %v", err)
	}
	status, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv status: %v", err)
	}
	if status.Status != consts.CtrlStatusSuccess {
		t.Fatalf("status = %#v, want success", status)
	}
	if got := runtime.lns.pipelines.Get(pipeline.Name); got == nil {
		t.Fatalf("pipeline %s was not started", pipeline.Name)
	}
}

func TestForwardStreamRegistryKeepsStreamUntilPipelineStop(t *testing.T) {
	listenerID := "forward-registry-listener"
	pipelineID := "forward-registry-pipeline"
	registry := newForwardStreamRegistry()
	lns := &listener{
		Name:        listenerID,
		IP:          "127.0.0.1",
		pipelines:   core.NewPipelines(),
		websites:    map[string]*Website{},
		pipelineRPC: &forwardPipelineRPC{listenerID: listenerID, registry: registry},
	}

	pipeline := &clientpb.Pipeline{
		Name:       pipelineID,
		ListenerId: listenerID,
		Enable:     true,
		Type:       consts.TCPPipeline,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       pipelineID,
				ListenerId: listenerID,
				Host:       "127.0.0.1",
				Port:       0,
			},
		},
		Tls:        &clientpb.TLS{},
		Encryption: []*clientpb.Encryption{},
		Secure:     &clientpb.Secure{},
	}
	started, err := lns.startPipeline(pipeline)
	if err != nil {
		t.Fatalf("startPipeline failed: %v", err)
	}
	t.Cleanup(func() { _ = started.Close() })

	beforeDisconnect := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID)
	taskStream := &mockTaskStream{
		ctx:     context.Background(),
		recvErr: io.EOF,
	}
	if err := beforeDisconnect.serve(taskStream); err != nil {
		t.Fatalf("serve EOF error = %v", err)
	}
	if afterDisconnect := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID); afterDisconnect != beforeDisconnect {
		t.Fatal("TaskStream disconnect should keep the existing registry stream")
	}

	status := lns.handleJobCtrl(&clientpb.JobCtrl{
		Id:   1,
		Ctrl: consts.CtrlPipelineStop,
		Job:  &clientpb.Job{Name: pipelineID, Pipeline: pipeline},
	})
	if status == nil || status.Status != consts.CtrlStatusSuccess {
		t.Fatalf("stop status = %#v, want success", status)
	}
	afterStop := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID)
	if afterStop == beforeDisconnect {
		t.Fatal("pipeline stop should remove the old registry stream")
	}
	select {
	case err := <-recvForwardStream(beforeDisconnect):
		if !errors.Is(err, io.EOF) {
			t.Fatalf("old stream Recv error = %v, want EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("old stream Recv did not unblock after pipeline stop")
	}
}

func TestForwardStreamRegistryRejectsOpenWhileRetiring(t *testing.T) {
	registry := newForwardStreamRegistry()
	oldStream := mustOpenForwardLocalStream(t, registry, "listener-retire", "pipeline-retire")

	release, err := registry.retire("listener-retire", "pipeline-retire")
	if err != nil {
		t.Fatalf("retire failed: %v", err)
	}
	if _, err := registry.open("listener-retire", "pipeline-retire"); !errors.Is(err, ErrForwardStreamClosing) {
		t.Fatalf("open while retiring error = %v, want ErrForwardStreamClosing", err)
	}
	select {
	case err := <-recvForwardStream(oldStream):
		if !errors.Is(err, io.EOF) {
			t.Fatalf("retired stream Recv error = %v, want EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("retired stream Recv did not unblock")
	}

	release()
	fresh, err := registry.open("listener-retire", "pipeline-retire")
	if err != nil {
		t.Fatalf("open after retirement failed: %v", err)
	}
	if fresh == oldStream {
		t.Fatal("open after retirement reused the retired stream")
	}
}

func TestForwardStreamRegistryBlocksReuseUntilTimedOutSendDrains(t *testing.T) {
	registry := newForwardStreamRegistry()
	listenerID := "listener-poisoned"
	pipelineID := "pipeline-poisoned"
	oldStream := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	remote := &barrierForwardTaskStream{
		ctx:         ctx,
		sendEntered: make(chan struct{}),
		releaseSend: make(chan struct{}),
	}
	serveResult := make(chan error, 1)
	go func() { serveResult <- oldStream.serve(remote) }()
	if err := oldStream.sendEvent(&clientpb.SpiteRequest{ListenerId: listenerID}); err != nil {
		t.Fatalf("sendEvent failed: %v", err)
	}
	select {
	case <-remote.sendEntered:
	case <-time.After(time.Second):
		t.Fatal("remote Send was not entered")
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer closeCancel()
	releaseRetirement, err := registry.retireContext(closeCtx, listenerID, pipelineID)
	if !errors.Is(err, ErrForwardStreamCloseTimeout) {
		t.Fatalf("retireContext error = %v, want ErrForwardStreamCloseTimeout", err)
	}
	releaseRetirement()
	if _, err := registry.open(listenerID, pipelineID); !errors.Is(err, ErrForwardStreamPoisoned) {
		t.Fatalf("open before old send drained error = %v, want ErrForwardStreamPoisoned", err)
	}

	close(remote.releaseSend)
	select {
	case err := <-serveResult:
		if err != nil {
			t.Fatalf("serve error after release = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not return after release")
	}

	deadline := time.Now().Add(time.Second)
	for {
		fresh, err := registry.open(listenerID, pipelineID)
		if err == nil {
			if fresh == oldStream {
				t.Fatal("registry reused the drained old stream")
			}
			break
		}
		if !errors.Is(err, ErrForwardStreamPoisoned) {
			t.Fatalf("open after drain error = %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatal("registry stayed poisoned after old send drained")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestForwardLocalStreamAllowsOnlyOneActiveAttach(t *testing.T) {
	stream := newForwardLocalStream(nil, "listener-attach", "pipeline-attach")

	releaseFirst, err := stream.attach()
	if err != nil {
		t.Fatalf("first attach failed: %v", err)
	}
	if _, err := stream.attach(); !errors.Is(err, ErrForwardStreamAlreadyAttached) {
		t.Fatalf("second active attach error = %v, want ErrForwardStreamAlreadyAttached", err)
	}

	releaseFirst()
	releaseSecond, err := stream.attach()
	if err != nil {
		t.Fatalf("attach after release failed: %v", err)
	}
	defer releaseSecond()

	// A stale release must not detach the newer owner.
	releaseFirst()
	if _, err := stream.attach(); !errors.Is(err, ErrForwardStreamAlreadyAttached) {
		t.Fatalf("attach after stale release error = %v, want ErrForwardStreamAlreadyAttached", err)
	}
}

func TestForwardTaskStreamRejectsSecondActiveAttach(t *testing.T) {
	registry := newForwardStreamRegistry()
	service := &forwardListenerService{registry: registry}
	listenerID := "listener-task-attach"
	pipelineID := "pipeline-task-attach"
	firstCtx, cancelFirst := context.WithCancel(metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"listener_id", listenerID,
		"pipeline_id", pipelineID,
	)))
	first := &mockTaskStream{ctx: firstCtx}
	firstResult := make(chan error, 1)
	go func() { firstResult <- service.TaskStream(first) }()

	deadline := time.Now().Add(time.Second)
	for {
		local := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID)
		local.attachMu.Lock()
		attached := local.attached
		local.attachMu.Unlock()
		if attached {
			break
		}
		if time.Now().After(deadline) {
			cancelFirst()
			t.Fatal("first TaskStream did not become active")
		}
		time.Sleep(time.Millisecond)
	}

	secondCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"listener_id", listenerID,
		"pipeline_id", pipelineID,
	))
	if err := service.TaskStream(&mockTaskStream{ctx: secondCtx}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("second TaskStream error = %v, want code %s", err, codes.AlreadyExists)
	}

	cancelFirst()
	select {
	case err := <-firstResult:
		if err != nil {
			t.Fatalf("first TaskStream error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first TaskStream did not stop after context cancellation")
	}
}

func TestForwardPipelineRPCDoesNotCreateStreamForCanceledContext(t *testing.T) {
	registry := newForwardStreamRegistry()
	rpc := &forwardPipelineRPC{listenerID: "listener-canceled", registry: registry}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := rpc.Register(ctx, &clientpb.RegisterSession{
		ListenerId: "listener-canceled",
		PipelineId: "pipeline-canceled",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Register error = %v, want context.Canceled", err)
	}

	registry.mu.Lock()
	streamCount := len(registry.streams)
	registry.mu.Unlock()
	if streamCount != 0 {
		t.Fatalf("registry stream count = %d, want 0", streamCount)
	}
}

func TestForwardStreamRegistryCleanupOnListenerClose(t *testing.T) {
	listenerID := "forward-close-listener"
	pipelineID := "forward-close-pipeline"
	registry := newForwardStreamRegistry()
	lns := &listener{
		Name:        listenerID,
		IP:          "127.0.0.1",
		pipelines:   core.NewPipelines(),
		websites:    map[string]*Website{},
		pipelineRPC: &forwardPipelineRPC{listenerID: listenerID, registry: registry},
	}

	pipeline := &clientpb.Pipeline{
		Name:       pipelineID,
		ListenerId: listenerID,
		Enable:     true,
		Type:       consts.CustomPipeline,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: pipelineID, ListenerId: listenerID},
		},
		Tls:        &clientpb.TLS{},
		Encryption: []*clientpb.Encryption{},
		Secure:     &clientpb.Secure{},
	}
	lns.pipelines.Add(NewCustomPipeline(pipeline))
	beforeClose := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID)

	if err := lns.Close(); err != nil {
		t.Fatalf("listener Close failed: %v", err)
	}
	afterClose := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID)
	if afterClose == beforeClose {
		t.Fatal("listener Close should remove the old registry stream")
	}
}

func recvForwardStream(stream *forwardLocalStream) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		_, err := stream.Recv()
		errCh <- err
	}()
	return errCh
}

func TestForwardListenerRejectsClientOnlyAuth(t *testing.T) {
	port := reserveForwardPort(t)
	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      "forward-listener-test",
		Auth:      writeClientOnlyForwardAuthConfig(t),
		IP:        "127.0.0.1",
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ListenHost: "127.0.0.1",
			ListenPort: port,
		},
	}
	_, err := NewForwardListener(cfg)
	if err == nil {
		t.Fatal("NewForwardListener succeeded with client-only auth, want serverAuth error")
	}
	if !strings.Contains(err.Error(), "serverAuth") {
		t.Fatalf("NewForwardListener error = %q, want serverAuth", err.Error())
	}
}
