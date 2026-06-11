package rpc

import (
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
)

type testListener struct{}

func (testListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }

func (testListener) Close() error { return nil }

func (testListener) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0} }

func TestRunClientListenerWrapsServeError(t *testing.T) {
	want := errors.New("serve failed")
	oldServe := serveClientGRPC
	serveClientGRPC = func(*grpc.Server, net.Listener) error {
		return want
	}
	defer func() { serveClientGRPC = oldServe }()

	err := runClientListener(grpc.NewServer(), testListener{})
	if !errors.Is(err, want) {
		t.Fatalf("runClientListener error = %v, want %v", err, want)
	}
}

func TestNormalizeClientListenerErrorTreatsStoppedServerAsNil(t *testing.T) {
	if err := normalizeClientListenerError(grpc.ErrServerStopped); err != nil {
		t.Fatalf("normalizeClientListenerError = %v, want nil", err)
	}
}

func TestNormalizeClientListenerErrorPreservesUnexpectedError(t *testing.T) {
	want := errors.New("serve failed")
	if err := normalizeClientListenerError(want); !errors.Is(err, want) {
		t.Fatalf("normalizeClientListenerError = %v, want %v", err, want)
	}
}
