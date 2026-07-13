package core

import (
	"errors"
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

func TestEventBrokerRejectsSlowSubscriber(t *testing.T) {
	broker := newTestBroker()
	sub := make(chan Event, eventBufSize)
	for i := 0; i < eventBufSize; i++ {
		sub <- Event{EventType: consts.EventBroadcast, Op: "queued"}
	}
	err := broker.dispatch(sub, Event{EventType: consts.EventBroadcast, Op: "overflow"})
	if !errors.Is(err, ErrEventSubscriberSlow) {
		t.Fatalf("dispatch error = %v, want ErrEventSubscriberSlow", err)
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
