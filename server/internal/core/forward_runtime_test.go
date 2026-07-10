package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/grpc"
)

type testForwardRPC struct {
	checkinCount atomic.Int32
}

func (r *testForwardRPC) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	r.checkinCount.Add(1)
	return &clientpb.Empty{}, nil
}

func (*testForwardRPC) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

type testForwardStream struct {
	sendErr error
}

func (s testForwardStream) Send(*clientpb.SpiteResponse) error {
	return s.sendErr
}

func (testForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	return nil, errors.New("not used")
}

type abortForwardStream struct {
	closeCalls atomic.Int32
}

func (*abortForwardStream) Send(*clientpb.SpiteResponse) error { return nil }
func (*abortForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	return nil, errors.New("not used")
}
func (s *abortForwardStream) CloseSend() error {
	s.closeCalls.Add(1)
	return nil
}

type abortForwardClient struct {
	*testForwardRPC
	stream *abortForwardStream
	ctx    context.Context
}

func (c *abortForwardClient) OpenForwardStream(ctx context.Context, _ Pipeline) (ForwardStream, error) {
	c.ctx = ctx
	return c.stream, nil
}

type testPipeline struct {
	id       string
	closeErr error
}

func (p testPipeline) ID() string { return p.id }

func (testPipeline) Start() error { return nil }

func (p testPipeline) Close() error { return p.closeErr }

func (p testPipeline) ToProtobuf() *clientpb.Pipeline {
	return &clientpb.Pipeline{Name: p.id}
}

func TestForwardAbortClosesStreamCancelsContextAndStopsHandler(t *testing.T) {
	stream := &abortForwardStream{}
	client := &abortForwardClient{testForwardRPC: &testForwardRPC{}, stream: stream}
	forward, err := NewForward(client, testPipeline{id: "pipe-abort"})
	if err != nil {
		t.Fatalf("NewForward failed: %v", err)
	}

	if err := forward.Abort(); err != nil {
		t.Fatalf("Abort failed: %v", err)
	}
	if err := forward.Abort(); err != nil {
		t.Fatalf("second Abort failed: %v", err)
	}
	if got := stream.closeCalls.Load(); got != 1 {
		t.Fatalf("CloseSend calls = %d, want 1", got)
	}
	select {
	case <-client.ctx.Done():
	default:
		t.Fatal("OpenForwardStream context was not canceled")
	}
	select {
	case <-forward.handlerDone:
	default:
		t.Fatal("forward handler was still running after Abort returned")
	}
	if got := Forwarders.Get(forward.RuntimeKey()); got != nil {
		t.Fatalf("aborted forward remained in registry: %#v", got)
	}
}

func TestForwardHandlerReturnsStreamSendError(t *testing.T) {
	want := errors.New("stream send failed")
	forward := &Forward{
		ctx:         context.Background(),
		Pipeline:    testPipeline{id: "pipe-a"},
		ListenerRpc: &testForwardRPC{},
		Stream:      testForwardStream{sendErr: want},
		implantC:    make(chan *Message, 1),
		done:        make(chan struct{}),
	}
	forward.alive.Store(true)

	forward.implantC <- &Message{
		SessionID:  "session-a",
		Spites:     &implantpb.Spites{Spites: []*implantpb.Spite{{Name: "exec"}}},
		RemoteAddr: "127.0.0.1:9000",
	}
	close(forward.implantC)

	err := forward.Handler()
	if !errors.Is(err, want) {
		t.Fatalf("Forward.Handler error = %v, want %v", err, want)
	}
}

func TestForwardersRemoveDeletesOnCloseError(t *testing.T) {
	want := errors.New("close failed")
	store := &forwarders{forwarders: &sync.Map{}}
	forward := &Forward{
		Pipeline: testPipeline{id: "pipe-remove", closeErr: want},
		Stream:   testForwardStream{},
		done:     make(chan struct{}),
	}
	forward.alive.Store(true)
	store.Add(forward)

	err := store.Remove(forward.ID())
	if !errors.Is(err, want) {
		t.Fatalf("Remove error = %v, want %v", err, want)
	}
	if got := store.Get(forward.ID()); got != nil {
		t.Fatalf("expected forwarder to be deleted, got %#v", got)
	}
}

func TestForwardHandlerCheckinCalledOncePerMessage(t *testing.T) {
	rpc := &testForwardRPC{}
	stream := &capturingForwardStream{}
	forward := &Forward{
		ctx:         context.Background(),
		Pipeline:    testPipeline{id: "pipe-checkin"},
		ListenerId:  "lns-checkin",
		ListenerRpc: rpc,
		Stream:      stream,
		implantC:    make(chan *Message, 1),
		done:        make(chan struct{}),
	}
	forward.alive.Store(true)

	forward.implantC <- &Message{
		SessionID: "sess-checkin",
		Spites: &implantpb.Spites{Spites: []*implantpb.Spite{
			{Name: "exec", TaskId: 1},
			{Name: "upload", TaskId: 2},
			{Name: "download", TaskId: 3},
		}},
		RemoteAddr: "10.0.0.1:9000",
	}
	close(forward.implantC)

	if err := forward.Handler(); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	got := rpc.checkinCount.Load()
	if got != 1 {
		t.Fatalf("Checkin called %d times for 1 message with 3 spites, want exactly 1", got)
	}
}

func TestForwardRawIDBytesUsesLittleEndian(t *testing.T) {
	got := forwardRawIDBytes(0x01020304)
	want := []byte{0x04, 0x03, 0x02, 0x01}
	if string(got) != string(want) {
		t.Fatalf("forwardRawIDBytes() = %v, want %v", got, want)
	}
}

type capturingForwardStream struct {
	mu       sync.Mutex
	captured []*clientpb.SpiteResponse
}

func (s *capturingForwardStream) Send(resp *clientpb.SpiteResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captured = append(s.captured, resp)
	return nil
}

func (s *capturingForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	return nil, errors.New("not used")
}

func (s *capturingForwardStream) responses() []*clientpb.SpiteResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]*clientpb.SpiteResponse, len(s.captured))
	copy(cp, s.captured)
	return cp
}

func TestForwardHandlerSetsListenerIdNotPipelineId(t *testing.T) {
	stream := &capturingForwardStream{}
	forward := &Forward{
		ctx:         context.Background(),
		Pipeline:    testPipeline{id: "pipeline-x"},
		ListenerId:  "listener-y",
		ListenerRpc: &testForwardRPC{},
		Stream:      stream,
		implantC:    make(chan *Message, 1),
		done:        make(chan struct{}),
	}
	forward.alive.Store(true)

	forward.implantC <- &Message{
		SessionID:  "sess-1",
		Spites:     &implantpb.Spites{Spites: []*implantpb.Spite{{Name: "exec", TaskId: 1}}},
		RemoteAddr: "10.0.0.1:8000",
	}
	close(forward.implantC)

	if err := forward.Handler(); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	responses := stream.responses()
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if responses[0].ListenerId != "listener-y" {
		t.Fatalf("SpiteResponse.ListenerId = %q, want %q", responses[0].ListenerId, "listener-y")
	}
}

func TestForwardersScopeDuplicatePipelineNamesByListener(t *testing.T) {
	store := &forwarders{forwarders: &sync.Map{}}
	forwardA := &Forward{
		Pipeline:   testPipeline{id: "shared-pipe"},
		ListenerId: "listener-a",
		Stream:     testForwardStream{},
		done:       make(chan struct{}),
	}
	forwardB := &Forward{
		Pipeline:   testPipeline{id: "shared-pipe"},
		ListenerId: "listener-b",
		Stream:     testForwardStream{},
		done:       make(chan struct{}),
	}
	forwardA.alive.Store(true)
	forwardB.alive.Store(true)

	store.Add(forwardA)
	store.Add(forwardB)

	if got := store.Get(PipelineRuntimeKey("listener-a", "shared-pipe")); got != forwardA {
		t.Fatalf("listener-a forwarder = %#v, want %#v", got, forwardA)
	}
	if got := store.Get(PipelineRuntimeKey("listener-b", "shared-pipe")); got != forwardB {
		t.Fatalf("listener-b forwarder = %#v, want %#v", got, forwardB)
	}
}
