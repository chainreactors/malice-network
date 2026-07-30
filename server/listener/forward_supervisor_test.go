package listener

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
)

type reconnectRecvResult struct {
	req *clientpb.SpiteRequest
	err error
}

type reconnectForwardStream struct {
	ctx     context.Context
	results chan reconnectRecvResult
	entered chan struct{}
	once    sync.Once
}

func newReconnectForwardStream(result reconnectRecvResult) *reconnectForwardStream {
	results := make(chan reconnectRecvResult, 1)
	results <- result
	return &reconnectForwardStream{results: results, entered: make(chan struct{})}
}

func newBlockingReconnectForwardStream() *reconnectForwardStream {
	return &reconnectForwardStream{
		results: make(chan reconnectRecvResult),
		entered: make(chan struct{}),
	}
}

func (*reconnectForwardStream) Send(*clientpb.SpiteResponse) error { return nil }

func (s *reconnectForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	s.once.Do(func() { close(s.entered) })
	select {
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	case result := <-s.results:
		return result.req, result.err
	}
}

type reconnectPipelineRPC struct {
	mu      sync.Mutex
	streams []*reconnectForwardStream
	opens   int
}

func (c *reconnectPipelineRPC) OpenForwardStream(ctx context.Context, _ core.Pipeline) (core.ForwardStream, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	index := c.opens
	c.opens++
	if index >= len(c.streams) {
		return nil, errors.New("no more forward streams")
	}
	stream := c.streams[index]
	stream.ctx = ctx
	return stream, nil
}

func TestForwardSupervisorCloseCancelsReconnectBackoff(t *testing.T) {
	oldForwarders := core.Forwarders
	oldConnections := core.Connections
	oldListenerSessions := core.ListenerSessions
	core.ResetTransientTransportState()
	t.Cleanup(func() {
		core.ResetTransientTransportState()
		core.Forwarders = oldForwarders
		core.Connections = oldConnections
		core.ListenerSessions = oldListenerSessions
	})

	first := newReconnectForwardStream(reconnectRecvResult{err: errors.New("connection reset")})
	rpc := &reconnectPipelineRPC{streams: []*reconnectForwardStream{first}}
	listener := &supervisorTestListener{}
	pipeline, err := NewHttpPipeline(rpc, testPipelineProtobuf("forward-close-backoff", consts.HTTPPipeline))
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	pipeline.stateMu.Lock()
	pipeline.Enable = true
	pipeline.srv = listener
	pipeline.stateMu.Unlock()
	forward, err := core.NewForward(rpc, pipeline)
	if err != nil {
		t.Fatalf("NewForward failed: %v", err)
	}
	core.Forwarders.Add(forward)
	pipeline.startForwardRecv(forward)

	deadline := time.Now().Add(time.Second)
	for rpc.openCount() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if rpc.openCount() < 2 {
		t.Fatalf("supervisor did not enter reconnect backoff, open count = %d", rpc.openCount())
	}

	started := time.Now()
	if err := pipeline.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("Close waited for reconnect backoff: %v", elapsed)
	}
	opensAfterClose := rpc.openCount()
	time.Sleep(350 * time.Millisecond)
	if got := rpc.openCount(); got != opensAfterClose {
		t.Fatalf("forward reconnected after Close: opens %d -> %d", opensAfterClose, got)
	}
	if got := listener.closed.Load(); got != 1 {
		t.Fatalf("public listener close count = %d, want 1", got)
	}
}

func (*reconnectPipelineRPC) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func (*reconnectPipelineRPC) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func (*reconnectPipelineRPC) GetArtifact(context.Context, *clientpb.Artifact, ...grpc.CallOption) (*clientpb.Artifact, error) {
	return &clientpb.Artifact{}, nil
}

func (c *reconnectPipelineRPC) openCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.opens
}

type supervisorTestListener struct {
	closed atomic.Int32
}

func (*supervisorTestListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (l *supervisorTestListener) Close() error {
	l.closed.Add(1)
	return nil
}
func (*supervisorTestListener) Addr() net.Addr { return testAddr("127.0.0.1:0") }

func TestForwardRecvResetReconnectsWithoutClosingPublicListener(t *testing.T) {
	oldForwarders := core.Forwarders
	oldConnections := core.Connections
	oldListenerSessions := core.ListenerSessions
	core.ResetTransientTransportState()
	t.Cleanup(func() {
		core.ResetTransientTransportState()
		core.Forwarders = oldForwarders
		core.Connections = oldConnections
		core.ListenerSessions = oldListenerSessions
	})

	for _, tc := range []struct {
		name  string
		start func(t *testing.T, rpc pipelineRPCClient, ln net.Listener) (core.Pipeline, func(*core.Forward), func() bool, func() error)
	}{
		{
			name: "http",
			start: func(t *testing.T, rpc pipelineRPCClient, ln net.Listener) (core.Pipeline, func(*core.Forward), func() bool, func() error) {
				pipeline, err := NewHttpPipeline(rpc, testPipelineProtobuf("forward-reconnect-http", consts.HTTPPipeline))
				if err != nil {
					t.Fatalf("NewHttpPipeline failed: %v", err)
				}
				pipeline.stateMu.Lock()
				pipeline.Enable = true
				pipeline.srv = ln
				pipeline.stateMu.Unlock()
				return pipeline, pipeline.startForwardRecv, pipeline.enabled, pipeline.Close
			},
		},
		{
			name: "tcp",
			start: func(t *testing.T, rpc pipelineRPCClient, ln net.Listener) (core.Pipeline, func(*core.Forward), func() bool, func() error) {
				pipeline, err := NewTcpPipeline(rpc, testPipelineProtobuf("forward-reconnect-tcp", consts.TCPPipeline))
				if err != nil {
					t.Fatalf("NewTcpPipeline failed: %v", err)
				}
				pipeline.stateMu.Lock()
				pipeline.Enable = true
				pipeline.ln = ln
				pipeline.stateMu.Unlock()
				return pipeline, pipeline.startForwardRecv, pipeline.enabled, pipeline.Close
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := newReconnectForwardStream(reconnectRecvResult{err: errors.New("connection reset")})
			second := newBlockingReconnectForwardStream()
			rpc := &reconnectPipelineRPC{streams: []*reconnectForwardStream{first, second}}
			listener := &supervisorTestListener{}
			pipeline, startRecv, enabled, closePipeline := tc.start(t, rpc, listener)
			forward, err := core.NewForward(rpc, pipeline)
			if err != nil {
				t.Fatalf("NewForward failed: %v", err)
			}
			core.Forwarders.Add(forward)
			startRecv(forward)
			t.Cleanup(func() {
				_ = closePipeline()
				_ = core.Forwarders.Remove(forward.RuntimeKey())
			})

			select {
			case <-second.entered:
			case <-time.After(2 * time.Second):
				t.Fatalf("replacement forward did not start, open count = %d", rpc.openCount())
			}
			if !enabled() {
				t.Fatal("forward reset disabled the public pipeline")
			}
			if got := listener.closed.Load(); got != 0 {
				t.Fatalf("public listener close count = %d, want 0", got)
			}
			if got := core.Forwarders.Get(forward.RuntimeKey()); got == nil || got == forward {
				t.Fatalf("forward registry was not replaced: %#v", got)
			}
		})
	}
}
