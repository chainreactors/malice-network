package listener

import (
	"context"
	"errors"
	"net"
	"net/http/httptest"
	"testing"

	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
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

func TestListenerHandlerReturnsJobStreamOpenError(t *testing.T) {
	want := errors.New("job stream open failed")
	oldOpen := openListenerJobStream
	openListenerJobStream = func(listenerrpc.ListenerRPCClient, context.Context) (listenerrpc.ListenerRPC_JobStreamClient, error) {
		return nil, want
	}
	defer func() { openListenerJobStream = oldOpen }()

	lns := &listener{Name: "listener-a"}
	err := lns.Handler()
	if !errors.Is(err, want) {
		t.Fatalf("listener.Handler error = %v, want %v", err, want)
	}
}
