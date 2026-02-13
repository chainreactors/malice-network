package core

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikoksr/notify"
)

// newTestBroker creates a minimal eventBroker for testing SafeGoWithTask.
// The publish channel is buffered so Publish() won't block.
// Caller reads from broker.publish directly to capture events.
func newTestBroker() *eventBroker {
	return &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    Notifier{notify: notify.New(), enable: false},
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
}

// ---------- SafeGo ----------

func TestSafeGo_NormalExecution(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		close(done)
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo fn did not execute within timeout")
	}
}

func TestSafeGo_PanicRecovered(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		defer close(done)
		panic("test panic")
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo did not recover from panic within timeout")
	}
}

func TestSafeGo_CleanupsExecuteInOrder(t *testing.T) {
	var mu sync.Mutex
	var order []int
	done := make(chan struct{})

	SafeGo(func() {
		// fn body — nothing special
	},
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
			close(done)
		},
	)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanups did not complete within timeout")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 {
		t.Fatalf("expected 3 cleanups, got %d", len(order))
	}
	// cleanups are registered via reverse-loop defer, so they execute in passed order: 1,2,3
	for i, v := range order {
		if v != i+1 {
			t.Errorf("cleanup order[%d] = %d, want %d", i, v, i+1)
		}
	}
}

func TestSafeGo_CleanupsRunOnPanic(t *testing.T) {
	var cleaned atomic.Int32
	done := make(chan struct{})

	SafeGo(func() {
		panic("boom")
	},
		func() { cleaned.Add(1) },
		func() { cleaned.Add(1) },
		func() {
			cleaned.Add(1)
			close(done)
		},
	)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanups did not run after panic within timeout")
	}

	if got := cleaned.Load(); got != 3 {
		t.Fatalf("expected 3 cleanups to run after panic, got %d", got)
	}
}

func TestSafeGo_CleanupPanicAlsoRecovered(t *testing.T) {
	done := make(chan struct{})

	SafeGo(func() {
		// normal fn
	},
		func() {
			panic("cleanup panic")
		},
		func() {
			// this cleanup runs before the panicking one (defer LIFO within order group).
			// The panicking cleanup will be caught by the outer recover.
			close(done)
		},
	)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("process should survive a panic inside cleanup")
	}
}

func TestSafeGo_NoCleanups(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		close(done)
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo without cleanups did not execute")
	}
}

// ---------- SafeGoWithInfo ----------

func TestSafeGoWithInfo_PanicRecovered(t *testing.T) {
	done := make(chan struct{})
	SafeGoWithInfo("test-goroutine", func() {
		defer close(done)
		panic("info panic")
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGoWithInfo did not recover from panic")
	}
}

func TestSafeGoWithInfo_CleanupsRunOnPanic(t *testing.T) {
	var cleaned atomic.Int32
	done := make(chan struct{})

	SafeGoWithInfo("test-goroutine", func() {
		panic("info boom")
	},
		func() { cleaned.Add(1) },
		func() {
			cleaned.Add(1)
			close(done)
		},
	)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanups did not run")
	}
	if got := cleaned.Load(); got != 2 {
		t.Fatalf("expected 2 cleanups, got %d", got)
	}
}

// ---------- SafeGoWithTask ----------

func TestSafeGoWithTask_PanicPublishesEvent(t *testing.T) {
	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	task := &Task{
		Id:        1,
		SessionId: "test-session",
		Type:      "test-type",
	}

	SafeGoWithTask(task, func() {
		panic("task panic")
	})

	// Read directly from the buffered publish channel (no Subscribe needed)
	select {
	case evt := <-broker.publish:
		if evt.Err == "" {
			t.Fatal("expected non-empty Err in event")
		}
		if evt.Task == nil {
			t.Fatal("expected Task in event")
		}
		if evt.Task.TaskId != task.Id {
			t.Errorf("event TaskId = %d, want %d", evt.Task.TaskId, task.Id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive error event from SafeGoWithTask panic")
	}
}

func TestSafeGoWithTask_NormalNoEvent(t *testing.T) {
	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	done := make(chan struct{})

	task := &Task{
		Id:        2,
		SessionId: "test-session",
		Type:      "test-type",
	}

	SafeGoWithTask(task, func() {
		close(done)
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGoWithTask fn did not execute")
	}

	// give a short window — no event should arrive
	select {
	case evt := <-broker.publish:
		t.Fatalf("unexpected event published on normal execution: %+v", evt)
	case <-time.After(200 * time.Millisecond):
		// good — no event
	}
}

func TestSafeGoWithTask_CleanupsRunOnPanic(t *testing.T) {
	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	// Wait for the recovery handler to finish publishing before restoring broker.
	// Recovery runs AFTER cleanups (defer LIFO), so we wait for the publish event.
	defer func() { EventBroker = oldBroker }()

	var cleaned atomic.Int32

	task := &Task{
		Id:        3,
		SessionId: "test-session",
		Type:      "test-type",
	}

	SafeGoWithTask(task, func() {
		panic("task boom")
	},
		func() { cleaned.Add(1) },
		func() { cleaned.Add(1) },
	)

	// Wait for the event published by the recovery handler — this guarantees
	// the entire goroutine (cleanups + recovery) has completed.
	select {
	case <-broker.publish:
	case <-time.After(2 * time.Second):
		t.Fatal("recovery handler did not publish event after task panic")
	}

	if got := cleaned.Load(); got != 2 {
		t.Fatalf("expected 2 cleanups, got %d", got)
	}
}

// ---------- Concurrency stress ----------

func TestSafeGo_ConcurrentPanics(t *testing.T) {
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		SafeGo(func() {
			defer wg.Done()
			panic("concurrent panic")
		})
	}

	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
		// all 100 goroutines panicked and recovered — process alive
	case <-time.After(5 * time.Second):
		t.Fatal("not all concurrent panicking goroutines recovered within timeout")
	}
}

// ---------- Real-world panic scenarios ----------

func TestSafeGo_SendOnClosedChannel(t *testing.T) {
	ch := make(chan int)
	close(ch)

	done := make(chan struct{})
	SafeGo(func() {
		defer close(done)
		ch <- 1 // send on closed channel
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo did not recover from send-on-closed-channel panic")
	}
}

func TestSafeGo_NilMapWrite(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		defer close(done)
		var m map[string]int
		m["key"] = 1
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo did not recover from nil map write panic")
	}
}

func TestSafeGo_IndexOutOfRange(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		defer close(done)
		s := []int{1, 2, 3}
		_ = s[10]
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo did not recover from index out of range panic")
	}
}

func TestSafeGo_NilPointerDeref(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		defer close(done)
		var p *int
		_ = *p
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo did not recover from nil pointer dereference")
	}
}

func TestSafeGo_TypeAssertionPanic(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		defer close(done)
		var i interface{} = "string"
		_ = i.(int)
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo did not recover from type assertion panic")
	}
}
