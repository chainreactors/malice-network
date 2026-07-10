package listener

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"google.golang.org/grpc/metadata"
)

func TestAuditForwardSendEventRejectsAfterClose(t *testing.T) {
	accepted := 0
	for range 64 {
		stream := newAuditForwardLocalStream(1, 1)
		stream.close()
		err := stream.sendEvent(&clientpb.SpiteRequest{ListenerId: "after-close"})
		if err == nil {
			accepted++
			continue
		}
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("sendEvent error = %v, want %v", err, io.ErrClosedPipe)
		}
	}

	// Both select branches are ready for a closed stream with an empty queue.
	// Missing all invalid accepts therefore has probability 2^-64.
	if accepted > 0 {
		t.Fatalf("sendEvent accepted %d of 64 events after close", accepted)
	}
}

func TestForwardSendEventClosedErrorNeverQueuesEvent(t *testing.T) {
	for i := 0; i < 2000; i++ {
		stream := newAuditForwardLocalStream(1, 1)
		start := make(chan struct{})
		errCh := make(chan error, 1)
		closedCh := make(chan struct{})
		go func() {
			<-start
			errCh <- stream.sendEvent(&clientpb.SpiteRequest{ListenerId: "racing-event"})
		}()
		go func() {
			<-start
			stream.close()
			close(closedCh)
		}()
		close(start)

		err := <-errCh
		<-closedCh
		if !errors.Is(err, io.ErrClosedPipe) {
			continue
		}
		select {
		case event := <-stream.events:
			t.Fatalf("iteration %d: sendEvent returned ErrClosedPipe after queueing %#v", i, event)
		default:
		}
	}
}

type barrierForwardTaskStream struct {
	ctx         context.Context
	sendEntered chan struct{}
	releaseSend chan struct{}
	enterOnce   sync.Once
	sendCount   atomic.Int32
}

func (s *barrierForwardTaskStream) Send(*clientpb.SpiteRequest) error {
	s.enterOnce.Do(func() { close(s.sendEntered) })
	<-s.releaseSend
	s.sendCount.Add(1)
	return nil
}

func (s *barrierForwardTaskStream) Recv() (*clientpb.SpiteRequest, error) {
	<-s.ctx.Done()
	return nil, io.EOF
}

func (s *barrierForwardTaskStream) SetHeader(metadata.MD) error  { return nil }
func (s *barrierForwardTaskStream) SendHeader(metadata.MD) error { return nil }
func (s *barrierForwardTaskStream) SetTrailer(metadata.MD)       {}
func (s *barrierForwardTaskStream) Context() context.Context     { return s.ctx }
func (s *barrierForwardTaskStream) SendMsg(interface{}) error    { return nil }
func (s *barrierForwardTaskStream) RecvMsg(interface{}) error    { return nil }

func TestForwardCloseWaitsForInFlightRemoteSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newAuditForwardLocalStream(1, 1)
	remote := &barrierForwardTaskStream{
		ctx:         ctx,
		sendEntered: make(chan struct{}),
		releaseSend: make(chan struct{}),
	}
	serveResult := make(chan error, 1)
	go func() { serveResult <- stream.serve(remote) }()

	if err := stream.sendEvent(&clientpb.SpiteRequest{ListenerId: "in-flight"}); err != nil {
		t.Fatalf("sendEvent failed: %v", err)
	}
	select {
	case <-remote.sendEntered:
	case <-time.After(time.Second):
		t.Fatal("remote Send was not entered")
	}

	closeReturned := make(chan struct{})
	go func() {
		stream.close()
		close(closeReturned)
	}()
	select {
	case <-closeReturned:
		t.Fatal("close returned while remote Send was still in flight")
	case <-time.After(25 * time.Millisecond):
	}

	close(remote.releaseSend)
	select {
	case <-closeReturned:
	case <-time.After(time.Second):
		t.Fatal("close did not return after remote Send completed")
	}
	if got := remote.sendCount.Load(); got != 1 {
		t.Fatalf("remote Send count = %d, want 1", got)
	}
	if err := stream.sendEvent(&clientpb.SpiteRequest{ListenerId: "after-close"}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("sendEvent after close = %v, want ErrClosedPipe", err)
	}

	select {
	case err := <-serveResult:
		if err != nil {
			t.Fatalf("serve error after close = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not return after close")
	}
}
