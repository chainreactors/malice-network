package core

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestEventBrokerRunDropsClosedSubscriber(t *testing.T) {
	broker := newTestBroker()
	closedSub := make(chan Event, 1)
	close(closedSub)
	goodSub := make(chan Event, 1)

	err := broker.dispatch(closedSub, Event{
		EventType: consts.EventBroadcast,
		Op:        "test",
		Message:   "boom",
	})
	if err == nil {
		t.Fatal("expected closed subscriber dispatch to fail")
	}

	want := Event{
		EventType: consts.EventBroadcast,
		Op:        "test",
		Message:   "boom",
	}
	if err := broker.dispatch(goodSub, want); err != nil {
		t.Fatalf("dispatch good subscriber error = %v", err)
	}

	select {
	case evt := <-goodSub:
		if evt.Op != want.Op {
			t.Fatalf("event op = %q, want %q", evt.Op, want.Op)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("healthy subscriber did not receive event")
	}
}

func TestEventBrokerRunStopClosesSubscribers(t *testing.T) {
	broker := newTestBroker()
	sub := make(chan Event, 1)
	resultCh := make(chan error, 1)

	go func() {
		resultCh <- broker.run()
	}()

	ready := make(chan struct{})
	broker.subscribe <- eventSubscription{events: sub, ready: ready}
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("subscriber was not registered")
	}
	close(broker.stop)

	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatalf("broker.run error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("broker.run did not stop")
	}

	_, ok := <-sub
	if ok {
		t.Fatal("expected subscriber channel to be closed")
	}
}

func TestEventBrokerStopIsIdempotent(t *testing.T) {
	broker := newTestBroker()
	broker.Stop()
	broker.Stop()
}

func TestEventBrokerStopCachesAcceptedImportantEvents(t *testing.T) {
	for i := 0; i < 100; i++ {
		broker := newTestBroker()
		result := make(chan error, 1)
		go func() { result <- broker.run() }()
		if err := broker.TryPublish(Event{
			EventType: consts.EventBroadcast,
			Op:        "accepted-before-stop",
			Important: true,
		}); err != nil {
			t.Fatalf("iteration %d: TryPublish failed: %v", i, err)
		}
		broker.Stop()
		select {
		case err := <-result:
			if err != nil {
				t.Fatalf("iteration %d: broker.run error: %v", i, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: broker.run did not stop", i)
		}
		if history := broker.GetAll(); len(history) != 1 || history[0].Op != "accepted-before-stop" {
			t.Fatalf("iteration %d: cached history = %#v", i, history)
		}
	}
}

func TestEventBrokerKeepsSlowLegacySubscriberConnected(t *testing.T) {
	broker := newTestBroker()
	sub := make(chan Event, eventBufSize)
	for i := 0; i < eventBufSize; i++ {
		sub <- Event{EventType: consts.EventBroadcast, Op: "queued"}
	}
	err := broker.dispatch(sub, Event{EventType: consts.EventBroadcast, Op: "overflow"})
	if err != nil {
		t.Fatalf("dispatch overflow error = %v, want nil", err)
	}
	for i := 0; i < eventBufSize; i++ {
		<-sub
	}
	want := Event{EventType: consts.EventBroadcast, Op: "recovered"}
	if err := broker.dispatch(sub, want); err != nil {
		t.Fatalf("dispatch after recovery error = %v", err)
	}
	if got := <-sub; got.Op != want.Op {
		t.Fatalf("event after recovery = %q, want %q", got.Op, want.Op)
	}
}

func TestEventBrokerTryPublishReturnsUnavailableWhenStopped(t *testing.T) {
	broker := newTestBroker()
	broker.managed.Store(true)

	err := broker.TryPublish(Event{
		EventType: consts.EventBroadcast,
		Op:        "test",
		Message:   "stopped",
	})
	if !errors.Is(err, ErrEventBrokerUnavailable) {
		t.Fatalf("TryPublish error = %v, want %v", err, ErrEventBrokerUnavailable)
	}
}

func TestEventBrokerStartSurvivesBrokenSubscriber(t *testing.T) {
	broker := newTestBroker()
	broker.Start()
	defer broker.Stop()

	deadline := time.After(2 * time.Second)
	for !broker.alive.Load() {
		select {
		case <-deadline:
			t.Fatal("broker did not become alive")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	closedSub := make(chan Event, 1)
	close(closedSub)
	closedReady := make(chan struct{})
	broker.subscribe <- eventSubscription{events: closedSub, ready: closedReady}
	<-closedReady
	if err := broker.TryPublish(Event{
		EventType: consts.EventBroadcast,
		Op:        "panic",
		Message:   "panic",
	}); err != nil {
		t.Fatalf("TryPublish panic trigger error = %v", err)
	}

	sub, err := broker.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe error = %v", err)
	}
	defer broker.Unsubscribe(sub)

	deadline = time.After(2 * time.Second)
subscribed:
	for {
		if err := broker.TryPublish(Event{
			EventType: consts.EventBroadcast,
			Op:        "ready",
			Message:   "ok",
		}); err != nil {
			t.Fatalf("TryPublish ready error = %v", err)
		}

		select {
		case evt, ok := <-sub:
			if ok && evt.Op == "ready" {
				break subscribed
			}
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			t.Fatal("subscriber did not become ready")
		}
	}

	if err := broker.TryPublish(Event{
		EventType: consts.EventBroadcast,
		Op:        "restarted",
		Message:   "ok",
	}); err != nil {
		t.Fatalf("TryPublish restarted error = %v", err)
	}

	deadline = time.After(2 * time.Second)
	for {
		select {
		case evt, ok := <-sub:
			if !ok {
				t.Fatal("subscriber channel unexpectedly closed")
			}
			if evt.Op == "restarted" {
				return
			}
		case <-deadline:
			t.Fatal("broker did not continue dispatching events")
		}
	}
}

func TestEventBrokerSubscribeWaitsForRegistration(t *testing.T) {
	broker := newTestBroker()
	returned := make(chan chan Event, 1)
	go func() {
		sub, _ := broker.Subscribe()
		returned <- sub
	}()

	select {
	case <-returned:
		t.Fatal("Subscribe returned before the broker registered the subscriber")
	case <-time.After(50 * time.Millisecond):
	}

	go func() {
		_ = broker.run()
	}()
	defer broker.Stop()

	var sub chan Event
	select {
	case sub = <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("Subscribe did not return after registration")
	}

	want := Event{EventType: consts.EventBroadcast, Op: "after-ready"}
	if err := broker.TryPublish(want); err != nil {
		t.Fatalf("TryPublish error = %v", err)
	}
	select {
	case got := <-sub:
		if got.Op != want.Op {
			t.Fatalf("event op = %q, want %q", got.Op, want.Op)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("registered subscriber missed the first event")
	}
}

func TestEventBrokerSubscribeWithHistoryHasNoPublishBoundaryGap(t *testing.T) {
	for i := 0; i < 100; i++ {
		broker := newTestBroker()
		go func() { _ = broker.run() }()

		boundary := Event{
			EventType: consts.EventBroadcast,
			Op:        "boundary",
			Important: true,
		}
		if err := broker.TryPublish(boundary); err != nil {
			broker.Stop()
			t.Fatalf("iteration %d: publish boundary event: %v", i, err)
		}
		events, history, err := broker.SubscribeWithHistory()
		if err != nil {
			broker.Stop()
			t.Fatalf("iteration %d: subscribe with history: %v", i, err)
		}
		marker := Event{EventType: consts.EventBroadcast, Op: "marker"}
		if err := broker.TryPublish(marker); err != nil {
			broker.Stop()
			t.Fatalf("iteration %d: publish marker: %v", i, err)
		}

		boundaryCount := 0
		for _, event := range history {
			if event.Op == boundary.Op {
				boundaryCount++
			}
		}
		for {
			select {
			case event, ok := <-events:
				if !ok {
					broker.Stop()
					t.Fatalf("iteration %d: subscription closed before marker", i)
				}
				if event.Op == boundary.Op {
					boundaryCount++
				}
				if event.Op == marker.Op {
					if boundaryCount != 1 {
						broker.Stop()
						t.Fatalf("iteration %d: boundary event count = %d, want 1", i, boundaryCount)
					}
					broker.Unsubscribe(events)
					broker.Stop()
					goto nextIteration
				}
			case <-time.After(2 * time.Second):
				broker.Stop()
				t.Fatalf("iteration %d: marker event not received", i)
			}
		}
	nextIteration:
	}
}

func TestEventBrokerSubscribeAfterStopReturnsUnavailable(t *testing.T) {
	broker := newTestBroker()
	broker.Stop()
	if _, err := broker.Subscribe(); !errors.Is(err, ErrEventBrokerUnavailable) {
		t.Fatalf("Subscribe error = %v, want %v", err, ErrEventBrokerUnavailable)
	}
}

func TestEventBrokerSubscribeRacingStopDoesNotHang(t *testing.T) {
	for i := 0; i < 100; i++ {
		broker := newTestBroker()
		go func() { _ = broker.run() }()
		result := make(chan error, 1)
		go func() {
			_, err := broker.Subscribe()
			result <- err
		}()
		broker.Stop()
		select {
		case <-result:
		case <-time.After(2 * time.Second):
			t.Fatalf("Subscribe hung while racing Stop at iteration %d", i)
		}
	}
}

func TestEventBrokerSubscribeIsReadyBeforeReturn(t *testing.T) {
	broker := newTestBroker()
	broker.Start()
	defer broker.Stop()

	for i := 0; i < 100; i++ {
		sub := mustSubscribeEvent(t, broker)
		event := Event{
			EventType: consts.EventBroadcast,
			Op:        fmt.Sprintf("ready-%d", i),
		}
		if err := broker.TryPublish(event); err != nil {
			t.Fatalf("TryPublish failed: %v", err)
		}
		select {
		case got := <-sub:
			if got.Op != event.Op {
				t.Fatalf("event op = %q, want %q", got.Op, event.Op)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d missed event published immediately after Subscribe", i)
		}
		broker.Unsubscribe(sub)
	}
}

func TestEventBrokerV2DeliversReadyThenOrderedEvents(t *testing.T) {
	broker := newTestBroker()
	broker.Start()
	defer broker.Stop()

	sub := mustSubscribeEventV2(t, broker, EventSubscription{})
	defer broker.UnsubscribeV2(sub)

	ready := waitSequencedEvent(t, sub)
	if !ready.Ready {
		t.Fatalf("first envelope ready = false, want true")
	}
	if ready.StreamID == "" {
		t.Fatal("ready envelope stream ID is empty")
	}

	for _, op := range []string{"first", "second"} {
		if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: op}); err != nil {
			t.Fatalf("TryPublish(%s) failed: %v", op, err)
		}
	}

	first := waitSequencedEvent(t, sub)
	second := waitSequencedEvent(t, sub)
	if first.Event.Op != "first" || second.Event.Op != "second" {
		t.Fatalf("event order = %q, %q, want first, second", first.Event.Op, second.Event.Op)
	}
	if first.Sequence+1 != second.Sequence {
		t.Fatalf("sequences = %d, %d, want consecutive values", first.Sequence, second.Sequence)
	}
	if first.OccurredAt.IsZero() || second.OccurredAt.IsZero() {
		t.Fatal("event timestamp is missing")
	}
}

func TestEventBrokerV2ReplaysEventsAfterCursor(t *testing.T) {
	broker := newTestBroker()
	broker.Start()
	defer broker.Stop()

	initial := mustSubscribeEventV2(t, broker, EventSubscription{})
	ready := waitSequencedEvent(t, initial)
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: "one", Important: true}); err != nil {
		t.Fatalf("TryPublish(one) failed: %v", err)
	}
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: "two", Important: true}); err != nil {
		t.Fatalf("TryPublish(two) failed: %v", err)
	}
	one := waitSequencedEvent(t, initial)
	_ = waitSequencedEvent(t, initial)
	broker.UnsubscribeV2(initial)

	barrier := mustSubscribeEvent(t, broker)
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: "three", Important: true}); err != nil {
		t.Fatalf("TryPublish(three) failed: %v", err)
	}
	_ = waitEvent(t, barrier)
	broker.Unsubscribe(barrier)

	reconnected := mustSubscribeEventV2(t, broker, EventSubscription{
		StreamID:      ready.StreamID,
		AfterSequence: one.Sequence,
		Replay:        true,
	})
	defer broker.UnsubscribeV2(reconnected)

	reconnectReady := waitSequencedEvent(t, reconnected)
	if !reconnectReady.Ready || reconnectReady.ResetRequired {
		t.Fatalf("reconnect ready = %#v, want ready without reset", reconnectReady)
	}
	replayedTwo := waitSequencedEvent(t, reconnected)
	replayedThree := waitSequencedEvent(t, reconnected)
	if !replayedTwo.Replayed || !replayedThree.Replayed {
		t.Fatal("replayed envelopes are not marked as replayed")
	}
	if replayedTwo.Event.Op != "two" || replayedThree.Event.Op != "three" {
		t.Fatalf("replayed ops = %q, %q, want two, three", replayedTwo.Event.Op, replayedThree.Event.Op)
	}
}

func TestEventBrokerV2SignalsResetWhenCursorFallsBehindHistory(t *testing.T) {
	broker := newTestBroker()
	broker.historyCapacity = 2
	broker.Start()
	defer broker.Stop()

	initial := mustSubscribeEventV2(t, broker, EventSubscription{})
	ready := waitSequencedEvent(t, initial)
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: "one", Important: true}); err != nil {
		t.Fatalf("TryPublish(one) failed: %v", err)
	}
	one := waitSequencedEvent(t, initial)
	broker.UnsubscribeV2(initial)

	barrier := mustSubscribeEvent(t, broker)
	for _, op := range []string{"two", "three", "four"} {
		if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: op, Important: true}); err != nil {
			t.Fatalf("TryPublish(%s) failed: %v", op, err)
		}
		_ = waitEvent(t, barrier)
	}
	broker.Unsubscribe(barrier)

	reconnected := mustSubscribeEventV2(t, broker, EventSubscription{
		StreamID:      ready.StreamID,
		AfterSequence: one.Sequence,
		Replay:        true,
	})
	defer broker.UnsubscribeV2(reconnected)

	reset := waitSequencedEvent(t, reconnected)
	if !reset.Ready || !reset.ResetRequired {
		t.Fatalf("ready/reset = %v/%v, want true/true", reset.Ready, reset.ResetRequired)
	}
	if reset.OldestSequence != 3 || reset.LatestSequence != 4 {
		t.Fatalf("history bounds = %d..%d, want 3..4", reset.OldestSequence, reset.LatestSequence)
	}
}

func TestEventBrokerV2HeartbeatsDoNotEvictReplayableEvents(t *testing.T) {
	broker := newTestBroker()
	broker.historyCapacity = 2
	broker.Start()
	defer broker.Stop()

	initial := mustSubscribeEventV2(t, broker, EventSubscription{})
	ready := waitSequencedEvent(t, initial)
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: "one", Important: true}); err != nil {
		t.Fatalf("TryPublish(one) failed: %v", err)
	}
	one := waitSequencedEvent(t, initial)
	broker.UnsubscribeV2(initial)

	barrier := mustSubscribeEvent(t, broker)
	for index := 0; index < 3; index++ {
		if err := broker.TryPublish(Event{EventType: consts.EventHeartbeat, Op: consts.CtrlHeartbeat1s}); err != nil {
			t.Fatalf("TryPublish(heartbeat %d) failed: %v", index, err)
		}
		_ = waitEvent(t, barrier)
	}
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: "two", Important: true}); err != nil {
		t.Fatalf("TryPublish(two) failed: %v", err)
	}
	_ = waitEvent(t, barrier)
	broker.Unsubscribe(barrier)

	reconnected := mustSubscribeEventV2(t, broker, EventSubscription{
		StreamID:      ready.StreamID,
		AfterSequence: one.Sequence,
		Replay:        true,
	})
	defer broker.UnsubscribeV2(reconnected)

	reconnectReady := waitSequencedEvent(t, reconnected)
	if reconnectReady.ResetRequired {
		t.Fatal("heartbeat-only sequence gap unexpectedly requires reset")
	}
	replayed := waitSequencedEvent(t, reconnected)
	if !replayed.Replayed || replayed.Event.Op != "two" {
		t.Fatalf("replayed event = %#v, want session event two", replayed)
	}
}

func TestEventBrokerV2TransientEventsDoNotEvictReplayableEvents(t *testing.T) {
	broker := newTestBroker()
	broker.historyCapacity = 2
	broker.Start()
	defer broker.Stop()

	initial := mustSubscribeEventV2(t, broker, EventSubscription{})
	ready := waitSequencedEvent(t, initial)
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: "one", Important: true}); err != nil {
		t.Fatalf("TryPublish(one) failed: %v", err)
	}
	one := waitSequencedEvent(t, initial)
	broker.UnsubscribeV2(initial)

	barrier := mustSubscribeEvent(t, broker)
	for index := 0; index < 5; index++ {
		if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: consts.CtrlSessionCheckin}); err != nil {
			t.Fatalf("TryPublish(checkin %d) failed: %v", index, err)
		}
		_ = waitEvent(t, barrier)
	}
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: "two", Important: true}); err != nil {
		t.Fatalf("TryPublish(two) failed: %v", err)
	}
	_ = waitEvent(t, barrier)
	broker.Unsubscribe(barrier)

	reconnected := mustSubscribeEventV2(t, broker, EventSubscription{
		StreamID:      ready.StreamID,
		AfterSequence: one.Sequence,
		Replay:        true,
	})
	defer broker.UnsubscribeV2(reconnected)

	reconnectReady := waitSequencedEvent(t, reconnected)
	if reconnectReady.ResetRequired {
		t.Fatal("transient event burst unexpectedly evicted replayable history")
	}
	replayed := waitSequencedEvent(t, reconnected)
	if !replayed.Replayed || replayed.Event.Op != "two" {
		t.Fatalf("replayed event = %#v, want durable event two", replayed)
	}
}

func TestEventBrokerV2ReplaysPersistedSessionUpdate(t *testing.T) {
	broker := newTestBroker()
	broker.Start()
	defer broker.Stop()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	initial := mustSubscribeEventV2(t, broker, EventSubscription{})
	ready := waitSequencedEvent(t, initial)
	broker.UnsubscribeV2(initial)

	barrier := mustSubscribeEvent(t, broker)
	session := newTestSession("session-update-replay")
	session.PushUpdate("note updated")
	if event := waitEvent(t, barrier); !event.Important {
		t.Fatal("persisted session update should be replayable")
	}
	broker.Unsubscribe(barrier)

	reconnected := mustSubscribeEventV2(t, broker, EventSubscription{
		StreamID:      ready.StreamID,
		AfterSequence: ready.Sequence,
		Replay:        true,
	})
	defer broker.UnsubscribeV2(reconnected)
	if reconnectReady := waitSequencedEvent(t, reconnected); reconnectReady.ResetRequired {
		t.Fatal("session update replay unexpectedly requires reset")
	}
	replayed := waitSequencedEvent(t, reconnected)
	if !replayed.Replayed || replayed.Event.Op != consts.CtrlSessionUpdate {
		t.Fatalf("replayed event = %#v, want session_update", replayed)
	}
}

func TestEventBrokerV2DroppedTransientEventDoesNotRequireReset(t *testing.T) {
	broker := newTestBroker()
	channel := make(chan SequencedEvent, 1)
	channel <- SequencedEvent{Ready: true}
	subscriber := newEventV2Subscriber(channel, EventSubscription{})

	err := broker.dispatchV2(subscriber, SequencedEvent{
		Sequence: 1,
		Event: &Event{
			EventType: consts.EventSession,
			Op:        consts.CtrlSessionCheckin,
		},
	})
	if err != nil {
		t.Fatalf("dispatchV2 failed: %v", err)
	}
	if subscriber.needsReset {
		t.Fatal("dropping a transient event should not require authoritative reset")
	}
}

func TestEventBrokerV2FiltersTopicsAndHeartbeats(t *testing.T) {
	broker := newTestBroker()
	broker.Start()
	defer broker.Stop()

	sub := mustSubscribeEventV2(t, broker, EventSubscription{Topics: []string{consts.EventSession}})
	defer broker.UnsubscribeV2(sub)
	_ = waitSequencedEvent(t, sub)

	if err := broker.TryPublish(Event{EventType: consts.EventHeartbeat, Op: consts.CtrlHeartbeat1s}); err != nil {
		t.Fatalf("TryPublish(heartbeat) failed: %v", err)
	}
	if err := broker.TryPublish(Event{EventType: consts.EventListener, Op: consts.CtrlListenerStart}); err != nil {
		t.Fatalf("TryPublish(listener) failed: %v", err)
	}
	if err := broker.TryPublish(Event{EventType: consts.EventSession, Op: consts.CtrlSessionRegister}); err != nil {
		t.Fatalf("TryPublish(session) failed: %v", err)
	}

	event := waitSequencedEvent(t, sub)
	if event.Event.EventType != consts.EventSession {
		t.Fatalf("event type = %q, want %q", event.Event.EventType, consts.EventSession)
	}
}

func mustSubscribeEvent(t *testing.T, broker *eventBroker) chan Event {
	t.Helper()
	events, err := broker.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	return events
}

func mustSubscribeEventV2(t *testing.T, broker *eventBroker, options EventSubscription) chan SequencedEvent {
	t.Helper()
	events, err := broker.SubscribeV2(options)
	if err != nil {
		t.Fatalf("SubscribeV2 failed: %v", err)
	}
	return events
}

func waitSequencedEvent(t *testing.T, ch <-chan SequencedEvent) SequencedEvent {
	t.Helper()
	select {
	case event := <-ch:
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sequenced event")
		return SequencedEvent{}
	}
}

func TestEventToProtobufWebsiteWithoutTLSDoesNotPanic(t *testing.T) {
	event := Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlWebsiteStart,
		Job: &clientpb.Job{
			Name: "site-no-tls",
			Pipeline: &clientpb.Pipeline{
				Name: "site-no-tls",
				Type: consts.WebsitePipeline,
				Body: &clientpb.Pipeline_Web{
					Web: &clientpb.Website{
						Name: "site-no-tls",
						Root: "/",
						Port: 8080,
					},
				},
			},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Event.ToProtobuf panicked for website without TLS: %v", r)
		}
	}()

	pb := event.ToProtobuf()
	if pb == nil {
		t.Fatal("Event.ToProtobuf returned nil")
	}
	if pb.Formatted == "" {
		t.Fatal("expected formatted event output")
	}
}

func TestEventToProtobufWebsiteContentAddNormalizesURL(t *testing.T) {
	event := Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlWebContentAdd,
		Job: &clientpb.Job{
			Name: "site-content",
			Pipeline: &clientpb.Pipeline{
				Name: "site-content",
				Type: consts.WebsitePipeline,
				Ip:   "192.168.239.110",
				Body: &clientpb.Pipeline_Web{
					Web: &clientpb.Website{
						Name: "site-content",
						Root: "/",
						Port: 8081,
					},
				},
			},
			Contents: map[string]*clientpb.WebContent{
				"/aaa": {
					Path: "/aaa",
				},
			},
		},
	}

	pb := event.ToProtobuf()
	if pb == nil {
		t.Fatal("Event.ToProtobuf returned nil")
	}
	want := "[job] web_content_add: content add success, path: http://192.168.239.110:8081/aaa"
	if pb.Formatted != want {
		t.Fatalf("formatted = %q, want %q", pb.Formatted, want)
	}
	if strings.Contains(pb.Formatted, ":8081//") {
		t.Fatalf("formatted contains duplicate slash: %q", pb.Formatted)
	}
}

func TestEventToProtobufSessionTaskWithoutSessionOrTaskDoesNotPanic(t *testing.T) {
	event := Event{
		EventType: consts.EventSession,
		Op:        consts.CtrlSessionTask,
		Message:   "task dispatched",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Event.ToProtobuf panicked for session task without session/task: %v", r)
		}
	}()

	pb := event.ToProtobuf()
	if pb == nil {
		t.Fatal("Event.ToProtobuf returned nil")
	}
	if pb.Formatted == "" {
		t.Fatal("expected formatted event output")
	}
	if pb.Formatted != "[unknown-session.0] run task unknown-task: task dispatched" {
		t.Fatalf("formatted = %q, want fallback session/task text", pb.Formatted)
	}
}
