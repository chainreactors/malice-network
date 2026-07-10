package listener

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"google.golang.org/grpc/metadata"
)

func newAuditForwardLocalStream(requestCapacity, eventCapacity int) *forwardLocalStream {
	return &forwardLocalStream{
		listenerID: "audit-listener",
		pipelineID: "audit-pipeline",
		requests:   make(chan *clientpb.SpiteRequest, requestCapacity),
		events:     make(chan *clientpb.SpiteRequest, eventCapacity),
		done:       make(chan struct{}),
	}
}

type auditRequestTaskStream struct {
	ctx       context.Context
	req       *clientpb.SpiteRequest
	delivered chan struct{}
	sent      bool
}

func (s *auditRequestTaskStream) Send(*clientpb.SpiteRequest) error { return nil }
func (s *auditRequestTaskStream) Recv() (*clientpb.SpiteRequest, error) {
	if !s.sent {
		s.sent = true
		close(s.delivered)
		return s.req, nil
	}
	<-s.ctx.Done()
	return nil, io.EOF
}
func (s *auditRequestTaskStream) SetHeader(metadata.MD) error  { return nil }
func (s *auditRequestTaskStream) SendHeader(metadata.MD) error { return nil }
func (s *auditRequestTaskStream) SetTrailer(metadata.MD)       {}
func (s *auditRequestTaskStream) Context() context.Context     { return s.ctx }
func (s *auditRequestTaskStream) SendMsg(interface{}) error    { return nil }
func (s *auditRequestTaskStream) RecvMsg(interface{}) error    { return nil }

func TestAuditForwardRegistryUsesBoundedQueues(t *testing.T) {
	stream := newForwardStreamRegistry().get("audit-listener", "audit-pipeline")
	if got := cap(stream.requests); got != 255 {
		t.Fatalf("request queue capacity = %d, want 255", got)
	}
	if got := cap(stream.events); got != 255 {
		t.Fatalf("event queue capacity = %d, want 255", got)
	}
}

func TestAuditForwardEventQueueBackpressurePreservesEvent(t *testing.T) {
	stream := newAuditForwardLocalStream(1, 2)
	first := &clientpb.SpiteRequest{ListenerId: "event-1"}
	second := &clientpb.SpiteRequest{ListenerId: "event-2"}
	third := &clientpb.SpiteRequest{ListenerId: "event-3"}
	stream.events <- first
	stream.events <- second

	started := make(chan struct{})
	resultCh := make(chan error, 1)
	go func() {
		close(started)
		resultCh <- stream.sendEvent(third)
	}()
	<-started

	select {
	case err := <-resultCh:
		t.Fatalf("sendEvent returned before queue capacity was released: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	if got := <-stream.events; got != first {
		t.Fatalf("first queued event = %#v, want %#v", got, first)
	}
	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatalf("sendEvent failed after capacity was released: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("sendEvent stayed blocked after capacity was released")
	}
	if got := <-stream.events; got != second {
		t.Fatalf("second queued event = %#v, want %#v", got, second)
	}
	if got := <-stream.events; got != third {
		t.Fatalf("backpressured event = %#v, want %#v", got, third)
	}
}

func TestAuditForwardEventQueueCloseUnblocksProducer(t *testing.T) {
	stream := newAuditForwardLocalStream(1, 1)
	stream.events <- &clientpb.SpiteRequest{ListenerId: "queue-full"}

	started := make(chan struct{})
	resultCh := make(chan error, 1)
	go func() {
		close(started)
		resultCh <- stream.sendEvent(&clientpb.SpiteRequest{ListenerId: "blocked-event"})
	}()
	<-started

	select {
	case err := <-resultCh:
		t.Fatalf("sendEvent returned before close: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	stream.close()
	select {
	case err := <-resultCh:
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("sendEvent error = %v, want %v", err, io.ErrClosedPipe)
		}
	case <-time.After(time.Second):
		t.Fatal("sendEvent stayed blocked after stream close")
	}
}

func TestAuditForwardRequestQueueBackpressurePreservesOrder(t *testing.T) {
	stream := newAuditForwardLocalStream(1, 1)
	first := &clientpb.SpiteRequest{ListenerId: "request-1"}
	second := &clientpb.SpiteRequest{ListenerId: "request-2"}
	stream.requests <- first

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	remote := &auditRequestTaskStream{
		ctx:       ctx,
		req:       second,
		delivered: make(chan struct{}),
	}
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- stream.serve(remote)
	}()

	select {
	case <-remote.delivered:
	case <-time.After(time.Second):
		t.Fatal("remote request was not received")
	}
	select {
	case err := <-serveResult:
		t.Fatalf("serve returned while request producer should be backpressured: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	if got, err := stream.Recv(); err != nil || got != first {
		t.Fatalf("first Recv = (%#v, %v), want (%#v, nil)", got, err, first)
	}
	secondResult := make(chan *clientpb.SpiteRequest, 1)
	go func() {
		got, _ := stream.Recv()
		secondResult <- got
	}()
	select {
	case got := <-secondResult:
		if got != second {
			t.Fatalf("second Recv = %#v, want %#v", got, second)
		}
	case <-time.After(time.Second):
		t.Fatal("request producer stayed blocked after capacity was released")
	}

	stream.close()
	select {
	case err := <-serveResult:
		if err != nil {
			t.Fatalf("serve error after close = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not return after close")
	}
	cancel()
}

func TestAuditForwardRequestQueueCloseUnblocksProducer(t *testing.T) {
	stream := newAuditForwardLocalStream(1, 1)
	stream.requests <- &clientpb.SpiteRequest{ListenerId: "queue-full"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	remote := &auditRequestTaskStream{
		ctx:       ctx,
		req:       &clientpb.SpiteRequest{ListenerId: "blocked-request"},
		delivered: make(chan struct{}),
	}
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- stream.serve(remote)
	}()

	select {
	case <-remote.delivered:
	case <-time.After(time.Second):
		t.Fatal("remote request was not received")
	}
	stream.close()

	select {
	case err := <-serveResult:
		if err != nil {
			t.Fatalf("serve error after close = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve stayed blocked after close")
	}
	cancel()
}
