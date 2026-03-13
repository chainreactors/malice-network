package core

import (
	"errors"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
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

	broker.subscribe <- sub
	deadline := time.After(2 * time.Second)
subscribed:
	for {
		broker.publish <- Event{
			EventType: consts.EventBroadcast,
			Op:        "ready",
			Message:   "ready",
		}
		select {
		case <-sub:
			break subscribed
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			t.Fatal("subscriber did not receive initial event")
		}
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
	broker.subscribe <- closedSub
	if err := broker.TryPublish(Event{
		EventType: consts.EventBroadcast,
		Op:        "panic",
		Message:   "panic",
	}); err != nil {
		t.Fatalf("TryPublish panic trigger error = %v", err)
	}

	sub := broker.Subscribe()
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
