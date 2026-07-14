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
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	remhelper "github.com/chainreactors/malice-network/helper/third/rem"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
)

var (
	errFirstStart  = errors.New("first start failed")
	errSecondStart = errors.New("second start failed")
)

type linearizedStartRPC struct {
	calls        atomic.Int32
	firstEntered chan struct{}
	releaseFirst chan struct{}
}

type committedStartStream struct {
	done chan struct{}
	once sync.Once
}

func newCommittedStartStream() *committedStartStream {
	return &committedStartStream{done: make(chan struct{})}
}

func (*committedStartStream) Send(*clientpb.SpiteResponse) error { return nil }
func (s *committedStartStream) Recv() (*clientpb.SpiteRequest, error) {
	<-s.done
	return nil, context.Canceled
}
func (s *committedStartStream) CloseSend() error {
	s.once.Do(func() { close(s.done) })
	return nil
}

type committedStartRPC struct {
	stream *committedStartStream
}

func (c *committedStartRPC) OpenForwardStream(context.Context, core.Pipeline) (core.ForwardStream, error) {
	return c.stream, nil
}
func (*committedStartRPC) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}
func (*committedStartRPC) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}
func (*committedStartRPC) GetArtifact(context.Context, *clientpb.Artifact, ...grpc.CallOption) (*clientpb.Artifact, error) {
	return nil, errors.New("artifact unavailable")
}

func (c *linearizedStartRPC) OpenForwardStream(context.Context, core.Pipeline) (core.ForwardStream, error) {
	if c.calls.Add(1) == 1 {
		close(c.firstEntered)
		<-c.releaseFirst
		return nil, errFirstStart
	}
	return nil, errSecondStart
}
func (*linearizedStartRPC) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}
func (*linearizedStartRPC) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}
func (*linearizedStartRPC) GetArtifact(context.Context, *clientpb.Artifact, ...grpc.CallOption) (*clientpb.Artifact, error) {
	return nil, errors.New("artifact unavailable")
}

func assertCloseWaitsForStart(t *testing.T, start, closeFn func() error, entered <-chan struct{}, release func()) {
	t.Helper()
	firstResult := make(chan error, 1)
	go func() { firstResult <- start() }()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first Start did not enter initialization")
	}

	closeResult := make(chan error, 1)
	go func() { closeResult <- closeFn() }()
	select {
	case err := <-closeResult:
		t.Fatalf("Close returned before the in-flight Start completed: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	secondResult := make(chan error, 1)
	go func() { secondResult <- start() }()
	select {
	case err := <-secondResult:
		t.Fatalf("second Start returned while the previous generation was still initializing: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	release()
	if err := <-firstResult; !errors.Is(err, errFirstStart) {
		t.Fatalf("first Start error = %v, want %v", err, errFirstStart)
	}
	if err := <-closeResult; err != nil {
		t.Fatalf("Close error = %v, want nil", err)
	}
	if err := <-secondResult; !errors.Is(err, errSecondStart) {
		t.Fatalf("waiting second Start error = %v, want %v", err, errSecondStart)
	}
	if err := start(); !errors.Is(err, errSecondStart) {
		t.Fatalf("post-Close Start error = %v, want %v", err, errSecondStart)
	}
}

func TestTCPPipelineCloseLinearizesWithInFlightStart(t *testing.T) {
	rpc := &linearizedStartRPC{firstEntered: make(chan struct{}), releaseFirst: make(chan struct{})}
	pipeline, err := NewTcpPipeline(rpc, &clientpb.Pipeline{
		Name: "linearized-tcp", Tls: &clientpb.TLS{}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Host: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	assertCloseWaitsForStart(t, pipeline.Start, pipeline.Close, rpc.firstEntered, func() { close(rpc.releaseFirst) })
}

func TestHTTPPipelineCloseLinearizesWithInFlightStart(t *testing.T) {
	rpc := &linearizedStartRPC{firstEntered: make(chan struct{}), releaseFirst: make(chan struct{})}
	pipeline, err := NewHttpPipeline(rpc, &clientpb.Pipeline{
		Name: "linearized-http", Tls: &clientpb.TLS{}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Host: "127.0.0.1", Params: "{}"}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	assertCloseWaitsForStart(t, pipeline.Start, pipeline.Close, rpc.firstEntered, func() { close(rpc.releaseFirst) })
}

func TestTCPPipelineStartsServingAfterForwardAndStateCommit(t *testing.T) {
	oldListen, oldCmux := tcpListen, tcpStartCmux
	ln := newRollbackListener()
	tcpListen = func(string, string) (net.Listener, error) { return ln, nil }
	t.Cleanup(func() { tcpListen, tcpStartCmux = oldListen, oldCmux })

	const listenerID = "committed-listener"
	const pipelineID = "committed-tcp"
	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	_ = core.Forwarders.Remove(key)
	t.Cleanup(func() { _ = core.Forwarders.Remove(key) })

	stream := newCommittedStartStream()
	pipeline, err := NewTcpPipeline(&committedStartRPC{stream: stream}, &clientpb.Pipeline{
		Name: pipelineID, ListenerId: listenerID, Tls: &clientpb.TLS{Enable: true}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Host: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	tcpStartCmux = func(got net.Listener, _ *tls.Config, _ func(net.Conn), _ core.GoErrorHandler) (net.Listener, error) {
		pipeline.stateMu.RLock()
		stateCommitted := pipeline.Enable && pipeline.starting && pipeline.ln == got
		pipeline.stateMu.RUnlock()
		if core.Forwarders.Get(key) == nil || !stateCommitted {
			t.Errorf("TCP serving started before forward/state commit: forward=%v stateCommitted=%t", core.Forwarders.Get(key) != nil, stateCommitted)
		}
		return got, nil
	}

	if err := pipeline.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

func TestHTTPPipelineStartsServingAfterForwardAndStateCommit(t *testing.T) {
	oldListen, oldCmux := httpListen, httpStartWithCmux
	ln := newRollbackListener()
	httpListen = func(string, string) (net.Listener, error) { return ln, nil }
	t.Cleanup(func() { httpListen, httpStartWithCmux = oldListen, oldCmux })

	const listenerID = "committed-listener"
	const pipelineID = "committed-http"
	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	_ = core.Forwarders.Remove(key)
	t.Cleanup(func() { _ = core.Forwarders.Remove(key) })

	stream := newCommittedStartStream()
	pipeline, err := NewHttpPipeline(&committedStartRPC{stream: stream}, &clientpb.Pipeline{
		Name: pipelineID, ListenerId: listenerID, Tls: &clientpb.TLS{Enable: true, Cert: &clientpb.Cert{}}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Host: "127.0.0.1", Params: "{}"}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	httpStartWithCmux = func(gotPipeline *HTTPPipeline, got net.Listener, _ *http.ServeMux) error {
		gotPipeline.stateMu.RLock()
		stateCommitted := gotPipeline.Enable && gotPipeline.starting && gotPipeline.srv == got
		gotPipeline.stateMu.RUnlock()
		if core.Forwarders.Get(key) == nil || !stateCommitted {
			t.Errorf("HTTP serving started before forward/state commit: forward=%v stateCommitted=%t", core.Forwarders.Get(key) != nil, stateCommitted)
		}
		return nil
	}

	if err := pipeline.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

func TestTCPPipelineServeFailureKeepsStartInFlight(t *testing.T) {
	oldListen, oldCmux := tcpListen, tcpStartCmux
	ln := newRollbackListener()
	tcpListen = func(string, string) (net.Listener, error) { return ln, nil }
	t.Cleanup(func() { tcpListen, tcpStartCmux = oldListen, oldCmux })

	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	tcpStartCmux = func(net.Listener, *tls.Config, func(net.Conn), core.GoErrorHandler) (net.Listener, error) {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
			return nil, errFirstStart
		}
		return nil, errSecondStart
	}
	pipeline, err := NewTcpPipeline(&committedStartRPC{stream: newCommittedStartStream()}, &clientpb.Pipeline{
		Name: "serve-failure-tcp", ListenerId: "serve-failure-listener", Tls: &clientpb.TLS{Enable: true}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Host: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Forwarders.Remove(core.PipelineRuntimeKey(pipeline.ListenerID, pipeline.ID())) })

	assertCloseWaitsForStart(t, pipeline.Start, pipeline.Close, entered, func() { close(release) })
}

func TestHTTPPipelineServeFailureKeepsStartInFlight(t *testing.T) {
	oldListen, oldCmux := httpListen, httpStartWithCmux
	ln := newRollbackListener()
	httpListen = func(string, string) (net.Listener, error) { return ln, nil }
	t.Cleanup(func() { httpListen, httpStartWithCmux = oldListen, oldCmux })

	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	httpStartWithCmux = func(*HTTPPipeline, net.Listener, *http.ServeMux) error {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
			return errFirstStart
		}
		return errSecondStart
	}
	pipeline, err := NewHttpPipeline(&committedStartRPC{stream: newCommittedStartStream()}, &clientpb.Pipeline{
		Name: "serve-failure-http", ListenerId: "serve-failure-listener", Tls: &clientpb.TLS{Enable: true, Cert: &clientpb.Cert{}}, Secure: &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Host: "127.0.0.1", Params: "{}"}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Forwarders.Remove(core.PipelineRuntimeKey(pipeline.ListenerID, pipeline.ID())) })

	assertCloseWaitsForStart(t, pipeline.Start, pipeline.Close, entered, func() { close(release) })
}

func TestREMCloseLinearizesWithInFlightStart(t *testing.T) {
	oldListen := remConsoleListen
	oldClose := remConsoleClose
	defer func() {
		remConsoleListen = oldListen
		remConsoleClose = oldClose
	}()

	entered := make(chan struct{})
	release := make(chan struct{})
	var listenCalls atomic.Int32
	var closeCalls atomic.Int32
	var firstListenFinished atomic.Bool
	var overlappingClose atomic.Bool
	remConsoleListen = func(*remhelper.RemConsole) error {
		if listenCalls.Add(1) == 1 {
			close(entered)
			<-release
			firstListenFinished.Store(true)
			return errFirstStart
		}
		return errSecondStart
	}
	remConsoleClose = func(*remhelper.RemConsole) error {
		if !firstListenFinished.Load() {
			overlappingClose.Store(true)
		}
		closeCalls.Add(1)
		return nil
	}

	pipeline := &REM{Name: "linearized-rem", con: &remhelper.RemConsole{}}
	assertCloseWaitsForStart(t, pipeline.Start, pipeline.Close, entered, func() { close(release) })
	if overlappingClose.Load() {
		t.Fatal("REM Close overlapped the in-flight Listen")
	}
	if got := closeCalls.Load(); got != 0 {
		t.Fatalf("REM close calls = %d, want 0 after failed starts", got)
	}
}

func TestREMCloseWaitsForSuccessfulListenCleanup(t *testing.T) {
	oldListen := remConsoleListen
	oldClose := remConsoleClose
	defer func() {
		remConsoleListen = oldListen
		remConsoleClose = oldClose
	}()

	entered := make(chan struct{})
	release := make(chan struct{})
	listenFinished := make(chan struct{})
	var closeCalls atomic.Int32
	var overlappingClose atomic.Bool
	remConsoleListen = func(*remhelper.RemConsole) error {
		close(entered)
		<-release
		close(listenFinished)
		return nil
	}
	remConsoleClose = func(*remhelper.RemConsole) error {
		select {
		case <-listenFinished:
		default:
			overlappingClose.Store(true)
		}
		closeCalls.Add(1)
		return nil
	}

	pipeline := &REM{Name: "linearized-rem-success", con: &remhelper.RemConsole{}}
	startResult := make(chan error, 1)
	go func() { startResult <- pipeline.Start() }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("REM Start did not enter Listen")
	}

	closeResult := make(chan error, 1)
	go func() { closeResult <- pipeline.Close() }()
	select {
	case err := <-closeResult:
		t.Fatalf("REM Close returned before Listen cleanup: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	if err := <-startResult; err != nil {
		t.Fatalf("canceled REM Start error = %v, want nil", err)
	}
	if err := <-closeResult; err != nil {
		t.Fatalf("REM Close error = %v, want nil", err)
	}
	if overlappingClose.Load() {
		t.Fatal("REM Console Close overlapped Listen")
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("REM Console Close calls = %d, want 1", got)
	}
	if err := pipeline.Close(); err != nil {
		t.Fatalf("repeated REM Close error = %v", err)
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("REM Console Close calls after repeated Close = %d, want 1", got)
	}
}
