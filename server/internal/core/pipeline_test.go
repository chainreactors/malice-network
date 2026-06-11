package core

import (
	"strings"
	"testing"
)

type nopReadWriteCloser struct{}

func (nopReadWriteCloser) Read([]byte) (int, error)    { return 0, nil }
func (nopReadWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (nopReadWriteCloser) Close() error                { return nil }

func TestPipelineConfigWrapConnRejectsNilReceiver(t *testing.T) {
	var pipeline *PipelineConfig

	_, err := pipeline.WrapConn(nopReadWriteCloser{})
	if err == nil {
		t.Fatal("WrapConn error = nil, want explicit nil pipeline config error")
	}
	if !strings.Contains(err.Error(), "pipeline config is nil") {
		t.Fatalf("WrapConn error = %v, want nil pipeline config message", err)
	}
}

func TestPipelineConfigWrapBindConnRejectsNilReceiver(t *testing.T) {
	var pipeline *PipelineConfig

	_, err := pipeline.WrapBindConn(nopReadWriteCloser{})
	if err == nil {
		t.Fatal("WrapBindConn error = nil, want explicit nil pipeline config error")
	}
	if !strings.Contains(err.Error(), "pipeline config is nil") {
		t.Fatalf("WrapBindConn error = %v, want nil pipeline config message", err)
	}
}
