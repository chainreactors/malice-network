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
	remHealthCheck = func(listenerrpc.ListenerRPCClient, context.Context, *clientpb.Pipeline) error {
		panic("health panic")
	}
	defer func() {
		remHealthCheck = oldHealthCheck
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
	oldBroker := core.EventBroker
	oldTicker := core.GlobalTicker
	defer func() {
		remHealthCheck = oldHealthCheck
		core.EventBroker = oldBroker
		core.GlobalTicker = oldTicker
	}()

	testTicker := core.NewTicker()
	defer testTicker.RemoveAll()
	core.GlobalTicker = testTicker

	broker := core.NewBroker()
	defer broker.Stop()
	sub, err := broker.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe error = %v", err)
	}
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
	rem := &REM{
		Name:           "rem-health",
		Enable:         true,
		ListenerID:     "listener-a",
		remConfig:      &clientpb.REM{},
		healthInterval: time.Millisecond,
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
				_ = rem.Close()
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

	_ = rem.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("healthLoop error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("healthLoop did not stop")
	}
}
