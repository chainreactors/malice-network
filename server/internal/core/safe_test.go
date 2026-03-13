package core

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	inotify "github.com/chainreactors/malice-network/server/internal/notify"
)

func newTestBroker() *eventBroker {
	return &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
}

func waitGuardedResult(t *testing.T, ch <-chan error) (error, bool) {
	t.Helper()
	select {
	case err, ok := <-ch:
		return err, ok
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for guarded result")
		return nil, false
	}
}

func waitEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case evt := <-ch:
		return evt
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

func TestRunGuarded_ReturnErrorPreserved(t *testing.T) {
	want := errors.New("primary failure")
	var handled error

	err := RunGuarded("return-error", func() error {
		return want
	}, func(err error) {
		handled = err
	})

	if !errors.Is(err, want) {
		t.Fatalf("RunGuarded error = %v, want %v", err, want)
	}
	if !errors.Is(handled, want) {
		t.Fatalf("handler error = %v, want %v", handled, want)
	}
}

func TestRunGuarded_PanicStringConvertedToPanicError(t *testing.T) {
	err := RunGuarded("panic-string", func() error {
		panic("boom")
	}, func(error) {})

	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected PanicError, got %T", err)
	}
	if panicErr.Label != "panic-string" {
		t.Fatalf("panic label = %q, want %q", panicErr.Label, "panic-string")
	}
	if panicErr.Recovered != "boom" {
		t.Fatalf("panic recovered = %#v, want %#v", panicErr.Recovered, "boom")
	}
	if len(panicErr.Stack) == 0 {
		t.Fatal("expected panic stack")
	}
}

func TestRunGuarded_PanicErrorPreservesCause(t *testing.T) {
	want := errors.New("explode")
	err := RunGuarded("panic-error", func() error {
		panic(want)
	}, func(error) {})

	if !errors.Is(err, want) {
		t.Fatalf("RunGuarded error = %v, want cause %v", err, want)
	}

	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected PanicError, got %T", err)
	}
	if panicErr.Cause == nil || panicErr.Cause.Error() != want.Error() {
		t.Fatalf("panic cause = %v, want %v", panicErr.Cause, want)
	}
}

func TestRunGuarded_CleanupsExecuteInOrderOnPanic(t *testing.T) {
	var mu sync.Mutex
	var order []int

	err := RunGuarded("cleanup-order", func() error {
		panic("cleanup-order-panic")
	}, func(error) {},
		func() {
			mu.Lock()
			order = append(order, 1)
			mu.Unlock()
		},
		func() {
			mu.Lock()
			order = append(order, 2)
			mu.Unlock()
		},
		func() {
			mu.Lock()
			order = append(order, 3)
			mu.Unlock()
		},
	)

	if err == nil {
		t.Fatal("expected panic error")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 {
		t.Fatalf("cleanup count = %d, want 3", len(order))
	}
	for i, got := range order {
		want := i + 1
		if got != want {
			t.Fatalf("cleanup order[%d] = %d, want %d", i, got, want)
		}
	}
}

func TestRunGuarded_CleanupPanicJoinedWithPrimaryError(t *testing.T) {
	want := errors.New("primary")
	var cleaned atomic.Int32

	err := RunGuarded("cleanup-join", func() error {
		return want
	}, func(error) {},
		func() { cleaned.Add(1) },
		func() { panic("cleanup boom") },
		func() { cleaned.Add(1) },
	)

	if !errors.Is(err, want) {
		t.Fatalf("expected primary error to be preserved, got %v", err)
	}
	if cleaned.Load() != 2 {
		t.Fatalf("cleanup count = %d, want 2", cleaned.Load())
	}

	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected cleanup panic to be joined, got %T", err)
	}
	if panicErr.Label != "cleanup-join cleanup[1]" {
		t.Fatalf("cleanup panic label = %q, want %q", panicErr.Label, "cleanup-join cleanup[1]")
	}
}

func TestRunGuarded_DestructiveRuntimePanics(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "send_on_closed_channel",
			fn: func() {
				ch := make(chan int)
				close(ch)
				ch <- 1
			},
		},
		{
			name: "nil_map_write",
			fn: func() {
				var m map[string]int
				m["key"] = 1
			},
		},
		{
			name: "index_out_of_range",
			fn: func() {
				_ = []int{1}[9]
			},
		},
		{
			name: "nil_pointer_deref",
			fn: func() {
				var p *int
				_ = *p
			},
		},
		{
			name: "bad_type_assertion",
			fn: func() {
				var value any = "string"
				_ = value.(int)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunGuarded(tt.name, func() error {
				tt.fn()
				return nil
			}, func(error) {})

			var panicErr *PanicError
			if !errors.As(err, &panicErr) {
				t.Fatalf("expected PanicError, got %T", err)
			}
			if len(panicErr.Stack) == 0 {
				t.Fatal("expected stack trace")
			}
		})
	}
}

func TestGoGuarded_ReturnsErrorAndInvokesHandlerOnce(t *testing.T) {
	want := errors.New("guarded failure")
	var calls atomic.Int32
	var handled error

	errCh := GoGuarded("go-guarded", func() error {
		return want
	}, func(err error) {
		calls.Add(1)
		handled = err
	})

	err, ok := waitGuardedResult(t, errCh)
	if !ok {
		t.Fatal("expected error result, channel closed early")
	}
	if !errors.Is(err, want) {
		t.Fatalf("GoGuarded error = %v, want %v", err, want)
	}
	if !errors.Is(handled, want) {
		t.Fatalf("handler error = %v, want %v", handled, want)
	}
	if calls.Load() != 1 {
		t.Fatalf("handler calls = %d, want 1", calls.Load())
	}

	_, ok = waitGuardedResult(t, errCh)
	if ok {
		t.Fatal("expected channel to be closed after the first result")
	}
}

func TestGoGuarded_ChannelClosesOnSuccess(t *testing.T) {
	done := make(chan struct{})

	errCh := GoGuarded("success", func() error {
		close(done)
		return nil
	}, func(error) {
		t.Fatal("handler must not run on success")
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("guarded function did not finish")
	}

	err, ok := waitGuardedResult(t, errCh)
	if ok || err != nil {
		t.Fatalf("expected closed channel without error, got ok=%v err=%v", ok, err)
	}
}

func TestGoGuarded_ConvertsPanicToError(t *testing.T) {
	done := make(chan struct{})

	errCh := GoGuarded("panic-bridge", func() error {
		defer close(done)
		panic("guarded panic")
	}, LogGuardedError("panic-bridge"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("guarded function did not run")
	}

	err, ok := waitGuardedResult(t, errCh)
	if !ok || err == nil {
		t.Fatal("expected GoGuarded to return an error")
	}
	if ErrorText(err) != "panic: guarded panic" {
		t.Fatalf("error text = %q, want %q", ErrorText(err), "panic: guarded panic")
	}
}

func TestGoGuarded_ConcurrentErrorStorm(t *testing.T) {
	const workers = 200
	var wg sync.WaitGroup
	var handled atomic.Int32
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		i := i
		GoGuarded(fmt.Sprintf("storm-%d", i), func() error {
			defer wg.Done()
			if i%2 == 0 {
				panic("storm panic")
			}
			return errors.New("storm error")
		}, func(error) {
			handled.Add(1)
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent guarded workers did not finish")
	}

	deadline := time.After(2 * time.Second)
	for handled.Load() != workers {
		select {
		case <-deadline:
			t.Fatalf("handler count = %d, want %d", handled.Load(), workers)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

