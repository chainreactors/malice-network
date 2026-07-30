package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/grpc"
)

type testForwardRPC struct {
	checkinCount atomic.Int32
}

var errForwardSessionMissing = errors.New("session missing")

type registerAwareForwardRPC struct {
	mu         sync.Mutex
	registered bool
	calls      []string
}

func (r *registerAwareForwardRPC) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "checkin")
	if !r.registered {
		return nil, errForwardSessionMissing
	}
	return &clientpb.Empty{}, nil
}

func (r *registerAwareForwardRPC) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "register")
	r.registered = true
	return &clientpb.Empty{}, nil
}

func (r *registerAwareForwardRPC) callOrder() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
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

type blockingCloseForwardStream struct {
	started chan struct{}
	release chan struct{}
	done    chan struct{}
}

func (*blockingCloseForwardStream) Send(*clientpb.SpiteResponse) error { return nil }
func (*blockingCloseForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	return nil, errors.New("not used")
}
func (s *blockingCloseForwardStream) CloseSend() error {
	defer close(s.done)
	close(s.started)
	<-s.release
	return nil
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

func TestForwardAbortContextReturnsWhenHandlerDoesNotStop(t *testing.T) {
	forward := &Forward{
		Pipeline:    testPipeline{id: "pipe-timeout"},
		Stream:      testForwardStream{},
		done:        make(chan struct{}),
		handlerDone: make(chan struct{}),
	}
	forward.alive.Store(true)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := forward.AbortContext(ctx); !errors.Is(err, ErrForwardShutdownTimeout) {
		t.Fatalf("AbortContext error = %v, want ErrForwardShutdownTimeout", err)
	}
}

func TestForwardAbortContextReturnsWhenCloseSendDoesNotStop(t *testing.T) {
	stream := &blockingCloseForwardStream{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
	forward := &Forward{
		Pipeline: testPipeline{id: "pipe-close-timeout"},
		Stream:   stream,
		done:     make(chan struct{}),
	}
	forward.alive.Store(true)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := forward.AbortContext(ctx)
	if !errors.Is(err, ErrForwardShutdownTimeout) {
		t.Fatalf("AbortContext error = %v, want ErrForwardShutdownTimeout", err)
	}
	select {
	case <-stream.started:
	default:
		t.Fatal("CloseSend was not called")
	}
	close(stream.release)
	select {
	case <-stream.done:
	case <-time.After(time.Second):
		t.Fatal("CloseSend did not finish after release")
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

func TestForwardersRemoveIfSamePreservesReplacement(t *testing.T) {
	store := &forwarders{forwarders: &sync.Map{}}
	oldForward := &Forward{
		Pipeline: testPipeline{id: "shared"},
		Stream:   testForwardStream{},
		done:     make(chan struct{}),
	}
	newForward := &Forward{
		Pipeline: testPipeline{id: "shared"},
		Stream:   testForwardStream{},
		done:     make(chan struct{}),
	}
	oldForward.alive.Store(true)
	newForward.alive.Store(true)
	store.Add(oldForward)
	store.Add(newForward)

	if err := store.removeIfSame("shared", oldForward); err != nil {
		t.Fatalf("removeIfSame returned error: %v", err)
	}
	if got := store.Get("shared"); got != newForward {
		t.Fatalf("replacement forward = %#v, want %#v", got, newForward)
	}
	select {
	case <-oldForward.done:
	default:
		t.Fatal("stale forward was not aborted")
	}
	select {
	case <-newForward.done:
		t.Fatal("replacement forward was aborted")
	default:
	}
}

func TestForwardersAddIfAbsentDoesNotReplaceCurrentGeneration(t *testing.T) {
	store := &forwarders{forwarders: &sync.Map{}}
	current := &Forward{
		Pipeline: testPipeline{id: "shared-generation"},
		Stream:   testForwardStream{},
		done:     make(chan struct{}),
	}
	staleReconnect := &Forward{
		Pipeline: testPipeline{id: "shared-generation"},
		Stream:   testForwardStream{},
		done:     make(chan struct{}),
	}
	current.alive.Store(true)
	staleReconnect.alive.Store(true)
	store.Add(current)

	if store.AddIfAbsent(staleReconnect) {
		t.Fatal("stale reconnect unexpectedly replaced the current generation")
	}
	if got := store.Get(current.RuntimeKey()); got != current {
		t.Fatalf("registered forward = %#v, want current generation %#v", got, current)
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

func TestForwardHandlerRegistersBeforeCheckinWithoutQueuingInit(t *testing.T) {
	oldConnections := Connections
	Connections = &connections{connections: &sync.Map{}}
	t.Cleanup(func() {
		Connections = oldConnections
	})

	const sessionID = "session-register-before-checkin"
	connection := &Connection{
		SessionID: sessionID,
		C:         make(chan *clientpb.SpiteRequest, 1),
	}
	connection.alive.Store(true)
	Connections.Add(connection)

	rpc := &registerAwareForwardRPC{}
	forward := &Forward{
		ctx:         context.Background(),
		Pipeline:    testPipeline{id: "pipe-register-order"},
		ListenerId:  "listener-register-order",
		ListenerRpc: rpc,
		Stream:      testForwardStream{},
		implantC:    make(chan *Message, 1),
		done:        make(chan struct{}),
	}
	forward.alive.Store(true)
	forward.implantC <- &Message{
		SessionID: sessionID,
		RawID:     0x01020304,
		Spites: &implantpb.Spites{
			Spites: []*implantpb.Spite{{
				Name: "register",
				Body: &implantpb.Spite_Register{Register: &implantpb.Register{
					Name: "new-agent",
				}},
			}},
		},
		RemoteAddr: "10.0.0.1:9000",
	}
	close(forward.implantC)

	if err := forward.Handler(); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	calls := rpc.callOrder()
	if len(calls) != 2 || calls[0] != "register" || calls[1] != "checkin" {
		t.Fatalf("RPC call order = %v, want [register checkin]", calls)
	}
	select {
	case req := <-connection.C:
		t.Fatalf("successful registration queued unexpected Init: %#v", req.GetSpite().GetInit())
	default:
	}
}

func TestForwardHandlerUnknownNonRegisterSessionQueuesInit(t *testing.T) {
	oldConnections := Connections
	Connections = &connections{connections: &sync.Map{}}
	t.Cleanup(func() {
		Connections = oldConnections
	})

	const (
		sessionID = "session-unknown-checkin"
		rawID     = uint32(0x01020304)
	)
	connection := &Connection{
		SessionID: sessionID,
		C:         make(chan *clientpb.SpiteRequest, 1),
	}
	connection.alive.Store(true)
	Connections.Add(connection)

	rpc := &registerAwareForwardRPC{}
	forward := &Forward{
		ctx:         context.Background(),
		Pipeline:    testPipeline{id: "pipe-unknown-checkin"},
		ListenerId:  "listener-unknown-checkin",
		ListenerRpc: rpc,
		Stream:      testForwardStream{},
		implantC:    make(chan *Message, 1),
		done:        make(chan struct{}),
	}
	forward.alive.Store(true)
	forward.implantC <- &Message{
		SessionID: sessionID,
		RawID:     rawID,
		Spites: &implantpb.Spites{
			Spites: []*implantpb.Spite{{
				Name: "ping",
				Body: &implantpb.Spite_Ping{Ping: &implantpb.Ping{}},
			}},
		},
	}
	close(forward.implantC)

	if err := forward.Handler(); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	select {
	case req := <-connection.C:
		init := req.GetSpite().GetInit()
		if init == nil {
			t.Fatalf("queued spite body = %T, want Init", req.GetSpite().Body)
		}
		if got, want := string(init.Data), string(forwardRawIDBytes(rawID)); got != want {
			t.Fatalf("Init raw ID bytes = %v, want %v", init.Data, forwardRawIDBytes(rawID))
		}
	default:
		t.Fatal("unknown non-register session did not queue Init")
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
