package listener

import (
	"errors"
	"net"
	"testing"
)

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
