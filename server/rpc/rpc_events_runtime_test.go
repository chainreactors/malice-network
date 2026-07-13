package rpc

import (
	"context"
	"errors"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type captureEventsStream struct {
	ctx         context.Context
	headerMu    sync.Mutex
	header      metadata.MD
	headerReady chan struct{}
	headerOnce  sync.Once
	events      chan *clientpb.Event
	sendGate    <-chan struct{}
	sendStarted chan struct{}
	sendOnce    sync.Once
}

func newCaptureEventsStream(ctx context.Context) *captureEventsStream {
	return &captureEventsStream{
		ctx:         ctx,
		headerReady: make(chan struct{}),
		events:      make(chan *clientpb.Event, 4),
	}
}

func (s *captureEventsStream) Send(event *clientpb.Event) error {
	if s.sendGate != nil {
		s.sendOnce.Do(func() { close(s.sendStarted) })
		select {
		case <-s.sendGate:
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}
	select {
	case s.events <- event:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *captureEventsStream) SetHeader(metadata.MD) error { return nil }

func (s *captureEventsStream) SendHeader(header metadata.MD) error {
	s.headerMu.Lock()
	s.header = header.Copy()
	s.headerMu.Unlock()
	s.headerOnce.Do(func() { close(s.headerReady) })
	return nil
}

func (s *captureEventsStream) SetTrailer(metadata.MD)    {}
func (s *captureEventsStream) Context() context.Context  { return s.ctx }
func (s *captureEventsStream) SendMsg(interface{}) error { return nil }
func (s *captureEventsStream) RecvMsg(interface{}) error { return io.EOF }

func (s *captureEventsStream) responseHeader() metadata.MD {
	s.headerMu.Lock()
	defer s.headerMu.Unlock()
	return s.header.Copy()
}

func newEventTestBroker(t *testing.T) interface {
	TryPublish(core.Event) error
	GetAll() []*core.Event
	Stop()
} {
	t.Helper()
	oldBroker := core.EventBroker
	broker := core.NewBroker()
	waitEventBrokerReady(t, broker)
	t.Cleanup(func() {
		broker.Stop()
		core.EventBroker = oldBroker
	})
	return broker
}

func waitHistorySize(t *testing.T, broker interface{ GetAll() []*core.Event }, size int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for len(broker.GetAll()) != size {
		if time.Now().After(deadline) {
			t.Fatalf("event history size = %d, want %d", len(broker.GetAll()), size)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestEventsReplaysHistoryWhenNegotiated(t *testing.T) {
	broker := newEventTestBroker(t)
	if err := broker.TryPublish(core.Event{
		EventType: consts.EventBroadcast,
		Op:        "history",
		Important: true,
	}); err != nil {
		t.Fatalf("publish history: %v", err)
	}
	waitHistorySize(t, broker, 1)

	baseContext, cancel := context.WithCancel(context.Background())
	ctx := metadata.NewIncomingContext(baseContext, metadata.Pairs(
		consts.EventStreamHistoryReplayHeader, "true",
	))
	stream := newCaptureEventsStream(ctx)
	result := make(chan error, 1)
	go func() { result <- NewServer().Events(&clientpb.Empty{}, stream) }()

	select {
	case event := <-stream.events:
		if event.Op != "history" {
			t.Fatalf("replayed event op = %q, want history", event.Op)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("negotiated event history was not replayed")
	}
	if values := stream.responseHeader().Get(consts.EventStreamHistoryCountHeader); len(values) != 1 || values[0] != "1" {
		t.Fatalf("history count header = %v, want [1]", values)
	}
	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Events returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Events did not stop after context cancellation")
	}
}

func TestEventsKeepsLegacyClientsOnLiveOnlyStream(t *testing.T) {
	broker := newEventTestBroker(t)
	if err := broker.TryPublish(core.Event{
		EventType: consts.EventBroadcast,
		Op:        "history",
		Important: true,
	}); err != nil {
		t.Fatalf("publish history: %v", err)
	}
	waitHistorySize(t, broker, 1)

	ctx, cancel := context.WithCancel(context.Background())
	stream := newCaptureEventsStream(ctx)
	result := make(chan error, 1)
	go func() { result <- NewServer().Events(&clientpb.Empty{}, stream) }()
	select {
	case <-stream.headerReady:
	case <-time.After(2 * time.Second):
		t.Fatal("legacy event stream header was not sent")
	}
	if values := stream.responseHeader().Get(consts.EventStreamHistoryCountHeader); len(values) != 0 {
		t.Fatalf("legacy history count header = %v, want absent", values)
	}
	if err := broker.TryPublish(core.Event{EventType: consts.EventBroadcast, Op: "live"}); err != nil {
		t.Fatalf("publish live event: %v", err)
	}
	select {
	case event := <-stream.events:
		if event.Op != "live" {
			t.Fatalf("legacy first event op = %q, want live", event.Op)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("legacy live event was not delivered")
	}
	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Events returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Events did not stop after context cancellation")
	}
}

func TestEventsReportsSubscriberClosureAsResyncError(t *testing.T) {
	broker := newEventTestBroker(t)
	stream := newCaptureEventsStream(context.Background())
	result := make(chan error, 1)
	go func() { result <- NewServer().Events(&clientpb.Empty{}, stream) }()
	select {
	case <-stream.headerReady:
	case <-time.After(2 * time.Second):
		t.Fatal("event stream header was not sent")
	}

	broker.Stop()
	select {
	case err := <-result:
		if status.Code(err) != codes.ResourceExhausted {
			t.Fatalf("Events status = %v, want ResourceExhausted: %v", status.Code(err), err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Events did not report subscriber closure")
	}
}

func TestEventsReportsRealSubscriberOverflow(t *testing.T) {
	broker := newEventTestBroker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newCaptureEventsStream(ctx)
	stream.events = make(chan *clientpb.Event, 256)
	stream.sendStarted = make(chan struct{})
	sendGate := make(chan struct{})
	stream.sendGate = sendGate
	result := make(chan error, 1)
	go func() { result <- NewServer().Events(&clientpb.Empty{}, stream) }()
	select {
	case <-stream.headerReady:
	case <-time.After(2 * time.Second):
		t.Fatal("event stream header was not sent")
	}

	const eventCount = 128
	for i := 0; i < eventCount; i++ {
		event := core.Event{
			EventType: consts.EventBroadcast,
			Op:        "overflow-" + strconv.Itoa(i),
			Important: true,
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			err := broker.TryPublish(event)
			if err == nil {
				break
			}
			if !errors.Is(err, core.ErrEventBrokerQueueFull) {
				t.Fatalf("publish overflow event %d: %v", i, err)
			}
			if time.Now().After(deadline) {
				t.Fatalf("publish overflow event %d stayed queue-full", i)
			}
			time.Sleep(time.Millisecond)
		}
	}
	select {
	case <-stream.sendStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("event stream Send did not block")
	}
	lastOp := "overflow-" + strconv.Itoa(eventCount-1)
	deadline := time.Now().Add(2 * time.Second)
	for {
		history := broker.GetAll()
		if len(history) > 0 && history[len(history)-1].Op == lastOp {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("broker did not process overflow burst")
		}
		time.Sleep(time.Millisecond)
	}

	close(sendGate)
	select {
	case err := <-result:
		if status.Code(err) != codes.ResourceExhausted {
			t.Fatalf("Events status = %v, want ResourceExhausted: %v", status.Code(err), err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Events did not report real subscriber overflow")
	}
	if len(stream.events) == 0 {
		t.Fatal("overflow test did not deliver any queued events")
	}
}
