package listener

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type testAddr string

func (a testAddr) Network() string { return "tcp" }

func (a testAddr) String() string { return string(a) }

type testListener struct {
	accept func() (net.Conn, error)
	close  func() error
	addr   net.Addr
}

func (l testListener) Accept() (net.Conn, error) {
	return l.accept()
}

func (l testListener) Close() error {
	if l.close != nil {
		return l.close()
	}
	return nil
}

func (l testListener) Addr() net.Addr {
	if l.addr != nil {
		return l.addr
	}
	return testAddr("127.0.0.1:0")
}

func TestAcceptConnLoopReturnsAcceptError(t *testing.T) {
	want := errors.New("cmux accept failed")
	err := acceptConnLoop("cmux test", testListener{
		accept: func() (net.Conn, error) {
			return nil, want
		},
	}, func(net.Conn) {})
	if !errors.Is(err, want) {
		t.Fatalf("acceptConnLoop error = %v, want %v", err, want)
	}
}

func TestHTTPPipelineHandlerRecoversAndWritesInternalServerError(t *testing.T) {
	pipeline := &HTTPPipeline{Name: "http-a"}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	resp := httptest.NewRecorder()

	pipeline.handler(resp, req)

	if resp.Code != 500 {
		t.Fatalf("status code = %d, want 500", resp.Code)
	}
}

func TestTCPPipelineStartAcceptLoopReturnsAcceptError(t *testing.T) {
	want := errors.New("accept failed")
	pipeline := &TCPPipeline{
		Name:   "tcp-a",
		Enable: true,
	}

	err := pipeline.startAcceptLoop(testListener{
		accept: func() (net.Conn, error) {
			return nil, want
		},
	}, "tcp pipeline")
	if !errors.Is(err, want) {
		t.Fatalf("startAcceptLoop error = %v, want %v", err, want)
	}
}

func TestListenerRunJobStreamOnceReturnsOpenError(t *testing.T) {
	want := errors.New("job stream open failed")
	oldOpen := openListenerJobStream
	openListenerJobStream = func(listenerrpc.ListenerRPCClient, context.Context) (listenerrpc.ListenerRPC_JobStreamClient, error) {
		return nil, want
	}
	defer func() { openListenerJobStream = oldOpen }()

	lns := &listener{Name: "listener-a"}
	_, err := lns.runJobStreamOnce()
	if !errors.Is(err, want) {
		t.Fatalf("runJobStreamOnce error = %v, want %v", err, want)
	}
}

type reconnectJobStream struct {
	ctx  context.Context
	recv func() (*clientpb.JobCtrl, error)
}

func (s *reconnectJobStream) Header() (metadata.MD, error) { return nil, nil }
func (s *reconnectJobStream) Trailer() metadata.MD         { return nil }
func (s *reconnectJobStream) CloseSend() error             { return nil }
func (s *reconnectJobStream) Context() context.Context     { return s.ctx }
func (s *reconnectJobStream) SendMsg(interface{}) error    { return nil }
func (s *reconnectJobStream) RecvMsg(interface{}) error    { return nil }
func (s *reconnectJobStream) Send(*clientpb.JobStatus) error {
	return nil
}
func (s *reconnectJobStream) Recv() (*clientpb.JobCtrl, error) {
	return s.recv()
}

var _ grpc.ClientStream = (*reconnectJobStream)(nil)
var _ listenerrpc.ListenerRPC_JobStreamClient = (*reconnectJobStream)(nil)

type registerListenerCaptureClient struct {
	listenerrpc.ListenerRPCClient
	request *clientpb.RegisterListener
}

func (c *registerListenerCaptureClient) RegisterListener(_ context.Context, request *clientpb.RegisterListener, _ ...grpc.CallOption) (*clientpb.Empty, error) {
	c.request = request
	return &clientpb.Empty{}, nil
}

type runtimeSnapshotPipeline struct {
	pipeline *clientpb.Pipeline
}

func (p *runtimeSnapshotPipeline) ID() string                     { return p.pipeline.GetName() }
func (p *runtimeSnapshotPipeline) Start() error                   { return nil }
func (p *runtimeSnapshotPipeline) Close() error                   { return nil }
func (p *runtimeSnapshotPipeline) ToProtobuf() *clientpb.Pipeline { return p.pipeline }

func TestReregisterListenerIncludesRuntimeSnapshot(t *testing.T) {
	rpcClient := &registerListenerCaptureClient{}
	lns := &listener{
		Rpc:       rpcClient,
		Name:      "listener-2",
		IP:        "127.0.0.1",
		pipelines: core.NewPipelines(),
		ctx:       context.Background(),
	}
	lns.pipelines.Add(&runtimeSnapshotPipeline{pipeline: &clientpb.Pipeline{
		Name:       "AA",
		ListenerId: "listener-2",
		Enable:     true,
	}})
	lns.pipelines.Add(&runtimeSnapshotPipeline{pipeline: &clientpb.Pipeline{
		Name:       "disabled-runtime",
		ListenerId: "listener-2",
		Enable:     false,
	}})
	lns.setWebsite("download", &Website{
		Name:   "download",
		Enable: true,
		PipelineConfig: &core.PipelineConfig{
			ListenerID: "listener-2",
		},
	})

	if err := reregisterListenerRPC(lns); err != nil {
		t.Fatalf("reregisterListenerRPC failed: %v", err)
	}
	if rpcClient.request == nil || rpcClient.request.Pipelines == nil {
		t.Fatal("re-registration did not include a runtime snapshot")
	}
	pipelines := rpcClient.request.Pipelines.GetPipelines()
	if len(pipelines) != 2 {
		t.Fatalf("runtime snapshot = %#v, want two pipelines", pipelines)
	}
	names := map[string]bool{}
	for _, pipeline := range pipelines {
		names[pipeline.GetName()] = true
		if pipeline.GetListenerId() != "listener-2" {
			t.Fatalf("runtime %q listener = %q, want listener-2", pipeline.GetName(), pipeline.GetListenerId())
		}
	}
	if !names["AA"] || !names["download"] {
		t.Fatalf("runtime snapshot names = %v, want AA and download", names)
	}
}

func TestListenerHandlerReregistersBeforeReopeningJobStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lns := &listener{
		Name:      "listener-reconnect",
		ctx:       ctx,
		cancel:    cancel,
		pipelines: core.NewPipelines(),
		websites:  make(map[string]*Website),
	}

	oldOpen := openListenerJobStream
	oldRegister := reregisterListenerRPC
	oldWait := waitListenerReconnect
	defer func() {
		openListenerJobStream = oldOpen
		reregisterListenerRPC = oldRegister
		waitListenerReconnect = oldWait
	}()

	events := make(chan string, 4)
	var opens atomic.Int32
	openListenerJobStream = func(listenerrpc.ListenerRPCClient, context.Context) (listenerrpc.ListenerRPC_JobStreamClient, error) {
		call := opens.Add(1)
		events <- "open"
		if call == 1 {
			return &reconnectJobStream{
				ctx: ctx,
				recv: func() (*clientpb.JobCtrl, error) {
					return nil, io.EOF
				},
			}, nil
		}
		return &reconnectJobStream{
			ctx: ctx,
			recv: func() (*clientpb.JobCtrl, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}, nil
	}
	reregisterListenerRPC = func(*listener) error {
		events <- "register"
		return nil
	}
	waitListenerReconnect = func(ctx context.Context, _ time.Duration) bool {
		return ctx.Err() == nil
	}

	done := make(chan error, 1)
	go func() {
		done <- lns.Handler()
	}()

	want := []string{"open", "register", "open"}
	for i, expected := range want {
		select {
		case got := <-events:
			if got != expected {
				t.Fatalf("event %d = %q, want %q", i, got, expected)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for reconnect event %d", i)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Handler returned error during shutdown: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Handler did not stop after listener cancellation")
	}
}
