package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc/metadata"
)

type testEventsV2Stream struct {
	ctx    context.Context
	header metadata.MD
	events chan *clientpb.EventEnvelope
}

func (stream *testEventsV2Stream) SetHeader(header metadata.MD) error {
	stream.header = metadata.Join(stream.header, header)
	return nil
}

func (stream *testEventsV2Stream) SendHeader(header metadata.MD) error {
	stream.header = metadata.Join(stream.header, header)
	return nil
}

func (stream *testEventsV2Stream) SetTrailer(metadata.MD)   {}
func (stream *testEventsV2Stream) Context() context.Context { return stream.ctx }
func (stream *testEventsV2Stream) SendMsg(message interface{}) error {
	return stream.Send(message.(*clientpb.EventEnvelope))
}
func (stream *testEventsV2Stream) RecvMsg(interface{}) error { return nil }
func (stream *testEventsV2Stream) Send(event *clientpb.EventEnvelope) error {
	stream.events <- event
	return nil
}

func TestEventsV2StreamsReadyAndSequencedEvents(t *testing.T) {
	newRPCTestEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	stream := &testEventsV2Stream{
		ctx:    ctx,
		events: make(chan *clientpb.EventEnvelope, 4),
	}
	result := make(chan error, 1)
	go func() {
		result <- (&Server{}).EventsV2(&clientpb.EventSubscription{
			Topics: []string{consts.EventSession},
		}, stream)
	}()

	ready := waitEventsV2Envelope(t, stream.events)
	if !ready.Ready || ready.StreamId == "" {
		t.Fatalf("ready envelope = %#v, want ready with stream ID", ready)
	}
	if got := stream.header.Get(consts.EventStreamReadyHeader); len(got) != 1 || got[0] != "2" {
		t.Fatalf("ready header = %v, want [2]", got)
	}

	if err := core.EventBroker.TryPublish(core.Event{
		EventType: consts.EventSession,
		Op:        consts.CtrlSessionRegister,
		Session:   &clientpb.Session{SessionId: "event-v2-session"},
	}); err != nil {
		t.Fatalf("TryPublish failed: %v", err)
	}

	event := waitEventsV2Envelope(t, stream.events)
	if event.Sequence == 0 || event.Event.GetSession().GetSessionId() != "event-v2-session" {
		t.Fatalf("event envelope = %#v, want sequenced session event", event)
	}

	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("EventsV2 returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("EventsV2 did not stop after context cancellation")
	}
}

func TestGetEventHonorsLatestEventLimit(t *testing.T) {
	_ = newRPCTestEnv(t)
	for _, message := range []string{"oldest", "middle", "latest"} {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			Message:   message,
			Important: true,
		})
	}

	limited, err := (&Server{}).GetEvent(context.Background(), &clientpb.Int{Limit: 2})
	if err != nil {
		t.Fatalf("GetEvent limited error: %v", err)
	}
	if got := len(limited.GetEvents()); got != 2 {
		t.Fatalf("limited event count = %d, want 2", got)
	}
	if got := string(limited.GetEvents()[0].GetMessage()); got != "middle" {
		t.Fatalf("first limited event = %q, want middle", got)
	}
	if got := string(limited.GetEvents()[1].GetMessage()); got != "latest" {
		t.Fatalf("last limited event = %q, want latest", got)
	}

	all, err := (&Server{}).GetEvent(context.Background(), &clientpb.Int{})
	if err != nil {
		t.Fatalf("GetEvent all error: %v", err)
	}
	if got := len(all.GetEvents()); got != 3 {
		t.Fatalf("unlimited event count = %d, want 3", got)
	}
}

func waitEventsV2Envelope(t *testing.T, events <-chan *clientpb.EventEnvelope) *clientpb.EventEnvelope {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventsV2 envelope")
		return nil
	}
}
