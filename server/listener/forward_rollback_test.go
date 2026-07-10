package listener

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
)

var errListenerInitialization = errors.New("listener initialization failed")

type rollbackForwardStream struct {
	closeCalls atomic.Int32
}

func (*rollbackForwardStream) Send(*clientpb.SpiteResponse) error { return nil }
func (*rollbackForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	return nil, errors.New("unexpected Recv")
}
func (s *rollbackForwardStream) CloseSend() error {
	s.closeCalls.Add(1)
	return nil
}

type rollbackPipelineRPC struct {
	stream *rollbackForwardStream
}

type rollbackListener struct {
	closed     chan struct{}
	closeOnce  sync.Once
	closeCalls atomic.Int32
}

func newRollbackListener() *rollbackListener {
	return &rollbackListener{closed: make(chan struct{})}
}

func (l *rollbackListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}
func (l *rollbackListener) Close() error {
	l.closeOnce.Do(func() {
		l.closeCalls.Add(1)
		close(l.closed)
	})
	return nil
}
func (*rollbackListener) Addr() net.Addr { return rollbackAddr("rollback") }

type rollbackAddr string

func (a rollbackAddr) Network() string { return string(a) }
func (a rollbackAddr) String() string  { return string(a) }

func (c *rollbackPipelineRPC) OpenForwardStream(context.Context, core.Pipeline) (core.ForwardStream, error) {
	return c.stream, nil
}
func (*rollbackPipelineRPC) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}
func (*rollbackPipelineRPC) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}
func (*rollbackPipelineRPC) GetArtifact(context.Context, *clientpb.Artifact, ...grpc.CallOption) (*clientpb.Artifact, error) {
	return nil, errors.New("artifact unavailable")
}

func assertForwardRolledBack(t *testing.T, pipeline core.Pipeline, stream *rollbackForwardStream, start func() error) {
	t.Helper()
	if err := start(); !errors.Is(err, errListenerInitialization) {
		t.Fatalf("Start error = %v, want %v", err, errListenerInitialization)
	}
	if got := stream.closeCalls.Load(); got != 1 {
		t.Fatalf("forward stream CloseSend calls = %d, want 1", got)
	}
	if got := core.Forwarders.Get(core.PipelineRuntimeKey(pipeline.ToProtobuf().ListenerId, pipeline.ID())); got != nil {
		t.Fatalf("uncommitted forward remained in registry: %#v", got)
	}
}

func assertRollbackState(t *testing.T, pipeline core.Pipeline, stream *rollbackForwardStream) {
	t.Helper()
	if got := stream.closeCalls.Load(); got != 1 {
		t.Fatalf("forward stream CloseSend calls = %d, want 1", got)
	}
	if got := core.Forwarders.Get(core.PipelineRuntimeKey(pipeline.ToProtobuf().ListenerId, pipeline.ID())); got != nil {
		t.Fatalf("uncommitted forward remained in registry: %#v", got)
	}
}

func TestTCPPipelineRollsBackForwardWhenListenFails(t *testing.T) {
	oldListen := tcpListen
	tcpListen = func(string, string) (net.Listener, error) { return nil, errListenerInitialization }
	t.Cleanup(func() { tcpListen = oldListen })

	stream := &rollbackForwardStream{}
	rpc := &rollbackPipelineRPC{stream: stream}
	pipeline, err := NewTcpPipeline(rpc, &clientpb.Pipeline{
		Name: "rollback-tcp", ListenerId: "rollback-listener", Tls: &clientpb.TLS{}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Host: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	assertForwardRolledBack(t, pipeline, stream, pipeline.Start)
}

func TestHTTPPipelineRollsBackForwardWhenListenFails(t *testing.T) {
	oldListen := httpListen
	httpListen = func(string, string) (net.Listener, error) { return nil, errListenerInitialization }
	t.Cleanup(func() { httpListen = oldListen })

	stream := &rollbackForwardStream{}
	rpc := &rollbackPipelineRPC{stream: stream}
	pipeline, err := NewHttpPipeline(rpc, &clientpb.Pipeline{
		Name: "rollback-http", ListenerId: "rollback-listener", Tls: &clientpb.TLS{}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Host: "127.0.0.1", Params: (&implanttypes.PipelineParams{}).String()}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	assertForwardRolledBack(t, pipeline, stream, pipeline.Start)
}

func TestTCPPipelineRollsBackForwardWhenCmuxFails(t *testing.T) {
	oldListen, oldCmux := tcpListen, tcpStartCmux
	ln := newRollbackListener()
	tcpListen = func(string, string) (net.Listener, error) { return ln, nil }
	tcpStartCmux = func(net.Listener, *tls.Config, func(net.Conn), core.GoErrorHandler) (net.Listener, error) {
		return nil, errListenerInitialization
	}
	t.Cleanup(func() { tcpListen, tcpStartCmux = oldListen, oldCmux })

	stream := &rollbackForwardStream{}
	pipeline, err := NewTcpPipeline(&rollbackPipelineRPC{stream: stream}, &clientpb.Pipeline{
		Name: "rollback-tcp-cmux", ListenerId: "rollback-listener", Tls: &clientpb.TLS{Enable: true}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Host: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	assertForwardRolledBack(t, pipeline, stream, pipeline.Start)
	if got := ln.closeCalls.Load(); got != 1 {
		t.Fatalf("listener Close calls = %d, want 1", got)
	}
}

func TestHTTPPipelineRollsBackForwardWhenCmuxFails(t *testing.T) {
	oldListen, oldCmux := httpListen, httpStartWithCmux
	ln := newRollbackListener()
	httpListen = func(string, string) (net.Listener, error) { return ln, nil }
	httpStartWithCmux = func(*HTTPPipeline, net.Listener, *http.ServeMux) error {
		return errListenerInitialization
	}
	t.Cleanup(func() { httpListen, httpStartWithCmux = oldListen, oldCmux })

	stream := &rollbackForwardStream{}
	pipeline, err := NewHttpPipeline(&rollbackPipelineRPC{stream: stream}, &clientpb.Pipeline{
		Name: "rollback-http-cmux", ListenerId: "rollback-listener", Tls: &clientpb.TLS{Enable: true, Cert: &clientpb.Cert{}}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Host: "127.0.0.1", Params: (&implanttypes.PipelineParams{}).String()}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	assertForwardRolledBack(t, pipeline, stream, pipeline.Start)
	if got := ln.closeCalls.Load(); got != 1 {
		t.Fatalf("listener Close calls = %d, want 1", got)
	}
}
