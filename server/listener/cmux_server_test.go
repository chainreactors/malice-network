package listener

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerLimitsHeaderReadTime(t *testing.T) {
	server := NewHTTPServer(http.NotFoundHandler())
	if server.ReadHeaderTimeout != httpReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", server.ReadHeaderTimeout, httpReadHeaderTimeout)
	}
	if server.ReadHeaderTimeout <= 0 {
		t.Fatal("ReadHeaderTimeout must be positive")
	}
}

func TestSplitTLSListenerLimitsClassifierReadTime(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mux, _, plain := splitTLSListener(listener, 20*time.Millisecond)
	serveDone := make(chan error, 1)
	go func() { serveDone <- mux.Serve() }()
	t.Cleanup(func() {
		_ = listener.Close()
		mux.Close()
		select {
		case err := <-serveDone:
			if err != nil && !errors.Is(err, net.ErrClosed) {
				t.Errorf("cmux serve: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("cmux Serve did not stop")
		}
	})

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}

	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		serverConn, err := plain.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- serverConn
	}()
	started := time.Now()
	select {
	case serverConn := <-accepted:
		_ = serverConn.Close()
		if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
			t.Fatalf("classifier timeout took %s", elapsed)
		}
	case err := <-acceptErr:
		t.Fatalf("plain listener accept: %v", err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("idle connection remained stuck in cmux classification")
	}
}
