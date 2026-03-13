package listener

import (
	"errors"
	"net"
	"testing"
)

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
