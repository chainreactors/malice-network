package listener

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	remhelper "github.com/chainreactors/malice-network/helper/third/rem"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func TestREMGetLinkFallsBackWhenRuntimePanics(t *testing.T) {
	rem := &REM{
		Name:      "rem-a",
		remConfig: &clientpb.REM{Link: "tcp://configured"},
		con:       &remhelper.RemConsole{},
	}

	if got := rem.getLink(); got != "tcp://configured" {
		t.Fatalf("getLink = %q, want %q", got, "tcp://configured")
	}
}

func TestREMHealthLoopPanicBecomesGuardedError(t *testing.T) {
	oldHealthCheck := remHealthCheck
	oldSleep := remSleep
	remHealthCheck = func(listenerrpc.ListenerRPCClient, context.Context, *clientpb.Pipeline) error {
		panic("health panic")
	}
	remSleep = func(time.Duration) {}
	defer func() {
		remHealthCheck = oldHealthCheck
		remSleep = oldSleep
	}()

	rem := &REM{
		Name:      "rem-b",
		Enable:    true,
		remConfig: &clientpb.REM{},
	}

	err := core.RunGuarded("rem-health", rem.healthLoop, func(error) {})
	var panicErr *core.PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected PanicError, got %T", err)
	}
}

func TestREMHealthLoopPublishesDegradedAndRecoveredEvents(t *testing.T) {
	oldHealthCheck := remHealthCheck
	oldSleep := remSleep
	oldBroker := core.EventBroker
	oldTicker := core.GlobalTicker
	defer func() {
		remHealthCheck = oldHealthCheck
		remSleep = oldSleep
		core.EventBroker = oldBroker
		core.GlobalTicker = oldTicker
	}()

	testTicker := core.NewTicker()
	defer testTicker.RemoveAll()
	core.GlobalTicker = testTicker

	broker := core.NewBroker()
	defer broker.Stop()
	sub := broker.Subscribe()
	defer broker.Unsubscribe(sub)

	readyDeadline := time.After(2 * time.Second)
	for {
		err := broker.TryPublish(core.Event{EventType: "test", Op: "ready"})
		if err == nil {
			break
		}
		if !errors.Is(err, core.ErrEventBrokerUnavailable) {
			t.Fatalf("unexpected broker readiness error: %v", err)
		}
		select {
		case <-readyDeadline:
			t.Fatal("broker did not become ready")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	var checks int
	remHealthCheck = func(listenerrpc.ListenerRPCClient, context.Context, *clientpb.Pipeline) error {
		checks++
		if checks <= 3 {
			return fmt.Errorf("failure-%d", checks)
		}
		return nil
	}
	remSleep = func(time.Duration) {}

	rem := &REM{
		Name:       "rem-health",
		Enable:     true,
		ListenerID: "listener-a",
		remConfig:  &clientpb.REM{},
	}

	done := make(chan error, 1)
	go func() {
		done <- rem.healthLoop()
	}()

	deadline := time.After(2 * time.Second)
	degraded := false
	recovered := false
	for !(degraded && recovered) {
		select {
		case evt := <-sub:
			switch evt.Op {
			case "health-check-failed":
				degraded = true
			case "health-check-recovered":
				recovered = true
				rem.Enable = false
			}
		case err := <-done:
			if err != nil {
				t.Fatalf("healthLoop error = %v, want nil", err)
			}
			if !(degraded && recovered) {
				t.Fatal("healthLoop exited before publishing both events")
			}
		case <-deadline:
			t.Fatal("timed out waiting for health events")
		}
	}

	rem.Enable = false
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("healthLoop error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("healthLoop did not stop")
	}
}
