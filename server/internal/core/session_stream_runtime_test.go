package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type testServerStream struct {
	ctx     context.Context
	sendMsg func(m interface{}) error
	recvMsg func(m interface{}) error
}

func (s *testServerStream) SetHeader(metadata.MD) error  { return nil }
func (s *testServerStream) SendHeader(metadata.MD) error { return nil }
func (s *testServerStream) SetTrailer(metadata.MD)       {}
func (s *testServerStream) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *testServerStream) SendMsg(m interface{}) error {
	if s.sendMsg != nil {
		return s.sendMsg(m)
	}
	return nil
}
func (s *testServerStream) RecvMsg(m interface{}) error {
	if s.recvMsg != nil {
		return s.recvMsg(m)
	}
	return nil
}

var _ grpc.ServerStream = (*testServerStream)(nil)

func TestSessionRequestWithStreamWriterReturnsErrorAfterSendFailure(t *testing.T) {
	sess := newTestSession("stream-writer")
	var sendCount atomic.Int32
	streamErr := errors.New("stream down")
	stream := &testServerStream{
		sendMsg: func(m interface{}) error {
			switch sendCount.Add(1) {
			case 1:
				return nil
			default:
				return streamErr
			}
		},
	}

	writer, respCh, err := sess.RequestWithStream(&clientpb.SpiteRequest{
		Session: sess.ToProtobufLite(),
		Task:    &clientpb.Task{TaskId: 9},
		Spite:   &implantpb.Spite{Name: "start"},
	}, stream, time.Second)
	if err != nil {
		t.Fatalf("RequestWithStream failed: %v", err)
	}

	if err := writer.Send(&implantpb.Spite{Name: "chunk-1"}); err != nil {
		t.Fatalf("first writer.Send failed unexpectedly: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		err = writer.Err()
		if errors.Is(err, streamErr) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("writer did not record stream error, got %v", err)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if err := writer.Send(&implantpb.Spite{Name: "chunk-2"}); !errors.Is(err, streamErr) {
		t.Fatalf("writer.Send error = %v, want %v", err, streamErr)
	}

	select {
	case _, ok := <-respCh:
		if ok {
			t.Fatal("response channel should be closed after send failure")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for response channel to close")
	}
}
