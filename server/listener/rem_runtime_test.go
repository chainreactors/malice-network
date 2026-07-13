package listener

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	remhelper "github.com/chainreactors/malice-network/helper/third/rem"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/rem/agent"
)

func setREMTestRunContext(rem *REM) context.Context {
	runCtx, cancel := context.WithCancel(context.Background())
	rem.stateMu.Lock()
	rem.runCtx = runCtx
	rem.runCancel = cancel
	rem.stateMu.Unlock()
	return runCtx
}

func TestREMStartClosePreservesConsoleCleanupError(t *testing.T) {
	oldListen := remConsoleListen
	oldClose := remConsoleClose
	defer func() {
		remConsoleListen = oldListen
		remConsoleClose = oldClose
	}()

	entered := make(chan struct{})
	release := make(chan struct{})
	cleanupErr := errors.New("console cleanup failed")
	remConsoleListen = func(*remhelper.RemConsole) error {
		close(entered)
		<-release
		return nil
	}
	remConsoleClose = func(*remhelper.RemConsole) error { return cleanupErr }

	rem := &REM{Name: "rem-cleanup", con: &remhelper.RemConsole{}}
	startResult := make(chan error, 1)
	go func() { startResult <- rem.Start() }()
	<-entered

	closeResult := make(chan error, 1)
	go func() { closeResult <- rem.Close() }()
	deadline := time.Now().Add(time.Second)
	for rem.enabled() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if rem.enabled() {
		t.Fatal("Close did not cancel the starting generation")
	}
	close(release)

	if err := <-startResult; !errors.Is(err, cleanupErr) {
		t.Fatalf("Start error = %v, want cleanup error", err)
	}
	if err := <-closeResult; !errors.Is(err, cleanupErr) {
		t.Fatalf("Close error = %v, want cleanup error", err)
	}
}

func TestREMCloseCancelsHealthRPCContextWithDeadline(t *testing.T) {
	oldHealthCheck := remHealthCheck
	defer func() { remHealthCheck = oldHealthCheck }()

	healthEntered := make(chan struct{})
	healthReturned := make(chan error, 1)
	releaseHealth := make(chan struct{})
	defer close(releaseHealth)
	var enteredOnce sync.Once
	remHealthCheck = func(_ listenerrpc.ListenerRPCClient, ctx context.Context, _ *clientpb.Pipeline) error {
		if _, ok := ctx.Deadline(); !ok {
			healthReturned <- errors.New("health context has no deadline")
			return nil
		}
		enteredOnce.Do(func() { close(healthEntered) })
		select {
		case <-ctx.Done():
			healthReturned <- ctx.Err()
		case <-releaseHealth:
			healthReturned <- errors.New("health check released without cancellation")
		}
		return nil
	}
	rem := &REM{Name: "rem-health-cancel", Enable: true, remConfig: &clientpb.REM{}}
	runCtx := setREMTestRunContext(rem)
	loopDone := make(chan error, 1)
	go func() { loopDone <- rem.healthLoopContext(runCtx) }()
	select {
	case <-healthEntered:
	case err := <-healthReturned:
		t.Fatalf("health check returned before Close: %v", err)
	case <-time.After(time.Second):
		t.Fatal("health check did not start")
	}

	if err := rem.Close(); err != nil {
		t.Fatalf("Close error = %v", err)
	}
	select {
	case err := <-healthReturned:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("health context error = %v, want context.Canceled", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Close did not cancel the in-flight health RPC")
	}
	select {
	case err := <-loopDone:
		if err != nil {
			t.Fatalf("health loop error = %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("health loop did not stop after Close")
	}
}

func TestREMAcceptErrorsUseInterruptibleBoundedBackoff(t *testing.T) {
	oldAccept := remConsoleAccept
	defer func() { remConsoleAccept = oldAccept }()

	calls := make(chan time.Time, 8)
	remConsoleAccept = func(*remhelper.RemConsole) (*agent.Agent, error) {
		calls <- time.Now()
		return nil, errors.New("temporary accept failure")
	}
	rem := &REM{
		Name:   "rem-accept-backoff",
		Enable: true,
	}
	runCtx := setREMTestRunContext(rem)
	done := make(chan error, 1)
	go func() { done <- rem.acceptLoopContextWithBackoff(runCtx, 20*time.Millisecond, 30*time.Millisecond) }()

	first := <-calls
	select {
	case second := <-calls:
		if delay := second.Sub(first); delay < 15*time.Millisecond {
			t.Fatalf("first retry delay = %v, want backoff", delay)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("second Accept did not occur within bounded backoff")
	}
	select {
	case <-calls:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("third Accept did not occur within maximum backoff")
	}

	startedClose := time.Now()
	if err := rem.Close(); err != nil {
		t.Fatalf("Close error = %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("acceptLoop error = %v", err)
		}
		if elapsed := time.Since(startedClose); elapsed > 100*time.Millisecond {
			t.Fatalf("acceptLoop cancellation took %v", elapsed)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Close did not interrupt Accept retry backoff")
	}
}

func TestREMAgentOwnershipDeletedWhenHandlerPanics(t *testing.T) {
	oldHandler := remConsoleHandler
	defer func() { remConsoleHandler = oldHandler }()
	remConsoleHandler = func(*remhelper.RemConsole, *agent.Agent) { panic("handler panic") }

	rem := &REM{Name: "rem-handler", con: &remhelper.RemConsole{}}
	ag := &agent.Agent{ID: "agent-panic"}
	rem.ownAgents.Store(ag.ID, struct{}{})
	err := core.RunGuarded("rem-agent", func() error {
		rem.handleAgent(ag)
		return nil
	}, func(error) {})
	var panicErr *core.PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("handler error = %v, want PanicError", err)
	}
	if _, ok := rem.ownAgents.Load(ag.ID); ok {
		t.Fatal("agent ownership remained after handler panic")
	}
}

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
	runCtx := setREMTestRunContext(rem)
	defer rem.Close()

	err := core.RunGuarded("rem-health", func() error {
		return rem.healthLoopContext(runCtx)
	}, func(error) {})
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
	runCtx := setREMTestRunContext(rem)
	go func() {
		done <- rem.healthLoopContext(runCtx)
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
