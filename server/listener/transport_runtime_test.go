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

type syncPipelineCaptureClient struct {
	listenerrpc.ListenerRPCClient
	pipelines []*clientpb.Pipeline
}

func (c *syncPipelineCaptureClient) SyncPipeline(_ context.Context, pipeline *clientpb.Pipeline, _ ...grpc.CallOption) (*clientpb.Empty, error) {
	c.pipelines = append(c.pipelines, pipeline)
	return &clientpb.Empty{}, nil
}

func TestRestoreRuntimeRegistrationsOnlySyncsActiveRuntimes(t *testing.T) {
	rpcClient := &syncPipelineCaptureClient{}
	lns := &listener{
		Rpc:  rpcClient,
		Name: "listener-reconnect",
	}

	err := lns.restoreRuntimeRegistrations(
		[]*clientpb.Pipeline{
			{Name: "tcp-active", Enable: true},
			{Name: "tcp-disabled", Enable: false},
		},
		[]*clientpb.Pipeline{
			{Name: "website-active", Enable: true},
		},
	)
	if err != nil {
		t.Fatalf("restoreRuntimeRegistrations returned error: %v", err)
	}
	if len(rpcClient.pipelines) != 2 {
		t.Fatalf("synced runtimes = %d, want 2", len(rpcClient.pipelines))
	}
	for _, pipeline := range rpcClient.pipelines {
		if pipeline.ListenerId != lns.ID() {
			t.Fatalf("runtime %q listener = %q, want %q", pipeline.Name, pipeline.ListenerId, lns.ID())
		}
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
	oldRegister := registerListenerRPC
	oldWait := waitListenerReconnect
	defer func() {
		openListenerJobStream = oldOpen
		registerListenerRPC = oldRegister
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
	registerListenerRPC = func(*listener) error {
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
