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
	closeErr   error
}

func (*rollbackForwardStream) Send(*clientpb.SpiteResponse) error { return nil }
func (*rollbackForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	return nil, errors.New("unexpected Recv")
}
func (s *rollbackForwardStream) CloseSend() error {
	s.closeCalls.Add(1)
	return s.closeErr
}

type rollbackPipelineRPC struct {
	stream *rollbackForwardStream
}

type rollbackListener struct {
	closed     chan struct{}
	closeOnce  sync.Once
	closeCalls atomic.Int32
	closeErr   error
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
	return l.closeErr
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

func addReplacementForward(t *testing.T, pipeline core.Pipeline, stream *rollbackForwardStream) *core.Forward {
	t.Helper()
	forward, err := core.NewForward(&rollbackPipelineRPC{stream: stream}, pipeline)
	if err != nil {
		t.Fatalf("create replacement forward: %v", err)
	}
	forward.ListenerId = pipeline.ToProtobuf().ListenerId
	core.Forwarders.Add(forward)
	return forward
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

func TestTCPPipelineStartFailureDiscardsLocalForwardStream(t *testing.T) {
	registry := newForwardStreamRegistry()
	listenerID := "rollback-local-listener"
	pipelineID := "rollback-local-tcp"
	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	var failedGeneration *forwardLocalStream
	oldListen := tcpListen
	tcpListen = func(string, string) (net.Listener, error) {
		registry.mu.Lock()
		failedGeneration = registry.streams[key]
		registry.mu.Unlock()
		return nil, errListenerInitialization
	}
	t.Cleanup(func() { tcpListen = oldListen })

	pipeline, err := NewTcpPipeline(&forwardPipelineRPC{listenerID: listenerID, registry: registry}, &clientpb.Pipeline{
		Name: pipelineID, ListenerId: listenerID, Tls: &clientpb.TLS{}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Host: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	if err := pipeline.Start(); !errors.Is(err, errListenerInitialization) {
		t.Fatalf("Start error = %v, want errListenerInitialization", err)
	}
	if failedGeneration == nil {
		t.Fatal("failed start did not create a forward stream")
	}
	if !failedGeneration.closed.Load() {
		t.Fatal("failed start left its forward stream open")
	}
	fresh := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID)
	if fresh == failedGeneration {
		t.Fatal("retry reused the failed forward stream generation")
	}
}

func TestTCPPipelineReturnsListenerAndForwardRollbackErrors(t *testing.T) {
	oldListen := tcpListen
	tcpListen = func(string, string) (net.Listener, error) { return nil, errListenerInitialization }
	t.Cleanup(func() { tcpListen = oldListen })

	errClose := errors.New("forward close failed")
	stream := &rollbackForwardStream{closeErr: errClose}
	pipeline, err := NewTcpPipeline(&rollbackPipelineRPC{stream: stream}, &clientpb.Pipeline{
		Name: "rollback-errors", ListenerId: "rollback-listener", Tls: &clientpb.TLS{}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Host: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	err = pipeline.Start()
	if !errors.Is(err, errListenerInitialization) || !errors.Is(err, errClose) {
		t.Fatalf("Start error = %v, want listener and rollback errors", err)
	}
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

func TestHTTPPipelineStartFailureDiscardsLocalForwardStream(t *testing.T) {
	registry := newForwardStreamRegistry()
	listenerID := "rollback-local-listener"
	pipelineID := "rollback-local-http"
	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	var failedGeneration *forwardLocalStream
	oldListen := httpListen
	httpListen = func(string, string) (net.Listener, error) {
		registry.mu.Lock()
		failedGeneration = registry.streams[key]
		registry.mu.Unlock()
		return nil, errListenerInitialization
	}
	t.Cleanup(func() { httpListen = oldListen })

	pipeline, err := NewHttpPipeline(&forwardPipelineRPC{listenerID: listenerID, registry: registry}, &clientpb.Pipeline{
		Name: pipelineID, ListenerId: listenerID, Tls: &clientpb.TLS{}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Host: "127.0.0.1", Params: (&implanttypes.PipelineParams{}).String()}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	if err := pipeline.Start(); !errors.Is(err, errListenerInitialization) {
		t.Fatalf("Start error = %v, want errListenerInitialization", err)
	}
	if failedGeneration == nil {
		t.Fatal("failed start did not create a forward stream")
	}
	if !failedGeneration.closed.Load() {
		t.Fatal("failed start left its forward stream open")
	}
	fresh := mustOpenForwardLocalStream(t, registry, listenerID, pipelineID)
	if fresh == failedGeneration {
		t.Fatal("retry reused the failed forward stream generation")
	}
}

func TestHTTPPipelineReturnsListenerAndForwardRollbackErrors(t *testing.T) {
	oldListen := httpListen
	httpListen = func(string, string) (net.Listener, error) { return nil, errListenerInitialization }
	t.Cleanup(func() { httpListen = oldListen })

	errClose := errors.New("forward close failed")
	stream := &rollbackForwardStream{closeErr: errClose}
	pipeline, err := NewHttpPipeline(&rollbackPipelineRPC{stream: stream}, &clientpb.Pipeline{
		Name: "rollback-http-errors", ListenerId: "rollback-listener", Tls: &clientpb.TLS{}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Host: "127.0.0.1", Params: (&implanttypes.PipelineParams{}).String()}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	err = pipeline.Start()
	if !errors.Is(err, errListenerInitialization) || !errors.Is(err, errClose) {
		t.Fatalf("Start error = %v, want listener and rollback errors", err)
	}
}

func TestTCPPipelineRollsBackForwardWhenCmuxFails(t *testing.T) {
	oldListen, oldCmux := tcpListen, tcpStartCmux
	errClose := errors.New("tcp listener close failed")
	ln := newRollbackListener()
	ln.closeErr = errClose
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
	err = pipeline.Start()
	if !errors.Is(err, errListenerInitialization) || !errors.Is(err, errClose) {
		t.Fatalf("Start error = %v, want initialization and listener close errors", err)
	}
	assertRollbackState(t, pipeline, stream)
	if got := ln.closeCalls.Load(); got != 1 {
		t.Fatalf("listener Close calls = %d, want 1", got)
	}
}

func TestHTTPPipelineRollsBackForwardWhenCmuxFails(t *testing.T) {
	oldListen, oldCmux := httpListen, httpStartWithCmux
	errClose := errors.New("http listener close failed")
	ln := newRollbackListener()
	ln.closeErr = errClose
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
	err = pipeline.Start()
	if !errors.Is(err, errListenerInitialization) || !errors.Is(err, errClose) {
		t.Fatalf("Start error = %v, want initialization and listener close errors", err)
	}
	assertRollbackState(t, pipeline, stream)
	if got := ln.closeCalls.Load(); got != 1 {
		t.Fatalf("listener Close calls = %d, want 1", got)
	}
}

func TestTCPPipelineFailedStartPreservesReplacementForward(t *testing.T) {
	oldListen, oldCmux := tcpListen, tcpStartCmux
	ln := newRollbackListener()
	tcpListen = func(string, string) (net.Listener, error) { return ln, nil }
	t.Cleanup(func() { tcpListen, tcpStartCmux = oldListen, oldCmux })

	stream := &rollbackForwardStream{}
	pipeline, err := NewTcpPipeline(&rollbackPipelineRPC{stream: stream}, &clientpb.Pipeline{
		Name: "rollback-tcp-replacement", ListenerId: "rollback-listener", Tls: &clientpb.TLS{Enable: true}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Host: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	key := core.PipelineRuntimeKey(pipeline.ListenerID, pipeline.ID())
	var replacement *core.Forward
	tcpStartCmux = func(net.Listener, *tls.Config, func(net.Conn), core.GoErrorHandler) (net.Listener, error) {
		replacement = addReplacementForward(t, pipeline, &rollbackForwardStream{})
		return nil, errListenerInitialization
	}
	t.Cleanup(func() { _ = core.Forwarders.Remove(key) })

	if err := pipeline.Start(); !errors.Is(err, errListenerInitialization) {
		t.Fatalf("Start error = %v, want %v", err, errListenerInitialization)
	}
	if got := core.Forwarders.Get(key); got != replacement {
		t.Fatalf("replacement forward = %#v, want %#v", got, replacement)
	}
	if got := stream.closeCalls.Load(); got != 1 {
		t.Fatalf("failed generation CloseSend calls = %d, want 1", got)
	}
}

func TestHTTPPipelineFailedStartPreservesReplacementForward(t *testing.T) {
	oldListen, oldCmux := httpListen, httpStartWithCmux
	ln := newRollbackListener()
	httpListen = func(string, string) (net.Listener, error) { return ln, nil }
	t.Cleanup(func() { httpListen, httpStartWithCmux = oldListen, oldCmux })

	stream := &rollbackForwardStream{}
	pipeline, err := NewHttpPipeline(&rollbackPipelineRPC{stream: stream}, &clientpb.Pipeline{
		Name: "rollback-http-replacement", ListenerId: "rollback-listener", Tls: &clientpb.TLS{Enable: true, Cert: &clientpb.Cert{}}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Host: "127.0.0.1", Params: (&implanttypes.PipelineParams{}).String()}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	key := core.PipelineRuntimeKey(pipeline.ListenerID, pipeline.ID())
	var replacement *core.Forward
	httpStartWithCmux = func(*HTTPPipeline, net.Listener, *http.ServeMux) error {
		replacement = addReplacementForward(t, pipeline, &rollbackForwardStream{})
		return errListenerInitialization
	}
	t.Cleanup(func() { _ = core.Forwarders.Remove(key) })

	if err := pipeline.Start(); !errors.Is(err, errListenerInitialization) {
		t.Fatalf("Start error = %v, want %v", err, errListenerInitialization)
	}
	if got := core.Forwarders.Get(key); got != replacement {
		t.Fatalf("replacement forward = %#v, want %#v", got, replacement)
	}
	if got := stream.closeCalls.Load(); got != 1 {
		t.Fatalf("failed generation CloseSend calls = %d, want 1", got)
	}
}
