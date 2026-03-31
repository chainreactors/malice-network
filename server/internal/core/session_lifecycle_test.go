package core

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	inotify "github.com/chainreactors/malice-network/server/internal/notify"
)

// ---------- Lifecycle: Ticker marks dead sessions ----------

func TestLifecycle_TickerMarksDead(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}

	// Add an expired session
	sess := newTestSession("dead-ticker",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Add(-10*time.Minute).Unix()),
	)
	s.Add(sess)

	// Simulate ticker callback
	tickerCallback := func() {
		for _, session := range s.All() {
			if !session.isAlived() {
				session.Save()
				session.Publish(consts.CtrlSessionDead,
					"session dead",
					true, true)
				s.Remove(session.ID)
			}
		}
	}
	tickerCallback()

	// Session should be removed from memory
	_, err := s.Get("dead-ticker")
	if err == nil {
		t.Fatal("expected dead session to be removed from memory")
	}

	// Context should be cancelled
	if sess.Ctx.Err() == nil {
		t.Fatal("expected session context to be cancelled after removal")
	}

	// Should have published CtrlSessionDead event
	select {
	case evt := <-broker.publish:
		if evt.Op != consts.CtrlSessionDead {
			t.Fatalf("expected CtrlSessionDead event, got %s", evt.Op)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive CtrlSessionDead event")
	}
}

func TestLifecycle_TickerKeepsAlive(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}

	// Add a recently-checked-in session
	sess := newTestSession("alive-ticker",
		withExpression("*/5 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)
	s.Add(sess)

	// Simulate ticker callback
	for _, session := range s.All() {
		if !session.isAlived() {
			session.Save()
			session.Publish(consts.CtrlSessionDead, "dead", true, true)
			s.Remove(session.ID)
		}
	}

	// Session should still be in memory
	got, err := s.Get("alive-ticker")
	if err != nil {
		t.Fatalf("alive session should remain in memory: %v", err)
	}
	if got.ID != "alive-ticker" {
		t.Fatalf("expected alive-ticker, got %s", got.ID)
	}

	// Context should still be alive
	if sess.Ctx.Err() != nil {
		t.Fatal("alive session context should not be cancelled")
	}

	// No event should be published
	select {
	case evt := <-broker.publish:
		t.Fatalf("unexpected event published for alive session: %+v", evt)
	case <-time.After(200 * time.Millisecond):
		// good — no event
	}
}

// ---------- Lifecycle: Get does not revive ----------

func TestLifecycle_GetDoesNotRevive(t *testing.T) {
	s := &sessions{active: &sync.Map{}}

	// Session is NOT in memory (simulates dead/DB-only session)
	_, err := s.Get("dead-session-123")
	if err == nil {
		t.Fatal("Get should return error for non-existent session")
	}

	// After Get fails, the session should still NOT be in memory
	_, err = s.Get("dead-session-123")
	if err == nil {
		t.Fatal("Get should still return error — no auto-recovery")
	}
}

// ---------- Lifecycle: Full cycle ----------

func TestLifecycle_RegisterCheckinDeadReborn(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize*4),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}

	// Step 1: Register — add session to memory
	sess := newTestSession("lifecycle-1",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)
	s.Add(sess)

	got, err := s.Get("lifecycle-1")
	if err != nil {
		t.Fatalf("session should be in memory after Add: %v", err)
	}
	if got.Ctx.Err() != nil {
		t.Fatal("new session context should be alive")
	}

	// Step 2: Checkin — update LastCheckin
	sess.SetLastCheckin(time.Now().Unix())
	sess.Save()

	// Step 3: Expire — set old LastCheckin
	sess.SetLastCheckin(time.Now().Add(-10 * time.Minute).Unix())

	// Step 4: Ticker detects dead
	for _, session := range s.All() {
		if !session.isAlived() {
			session.Save()
			session.Publish(consts.CtrlSessionDead, "dead", true, true)
			s.Remove(session.ID)
		}
	}

	_, err = s.Get("lifecycle-1")
	if err == nil {
		t.Fatal("dead session should be removed from memory")
	}
	if sess.Ctx.Err() == nil {
		t.Fatal("dead session context should be cancelled")
	}

	// Step 5: Reborn — create new session with same ID (simulating Checkin recovery)
	rebornSess := newTestSession("lifecycle-1",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)
	s.Add(rebornSess)

	got, err = s.Get("lifecycle-1")
	if err != nil {
		t.Fatalf("reborn session should be in memory: %v", err)
	}
	if got.Ctx.Err() != nil {
		t.Fatal("reborn session should have a fresh context")
	}

	// Old session context should still be cancelled
	if sess.Ctx.Err() == nil {
		t.Fatal("original session context should remain cancelled")
	}
}

// ---------- Lifecycle: Remove and Re-Add ----------

func TestLifecycle_RemoveAndReAdd(t *testing.T) {
	s := &sessions{active: &sync.Map{}}

	sess1 := newTestSession("reuse-1")
	s.Add(sess1)

	s.Remove("reuse-1")

	if sess1.Ctx.Err() == nil {
		t.Fatal("removed session context should be cancelled")
	}

	// Add a new session with same ID
	sess2 := newTestSession("reuse-1")
	s.Add(sess2)

	got, err := s.Get("reuse-1")
	if err != nil {
		t.Fatalf("re-added session should be accessible: %v", err)
	}
	if got.Ctx.Err() != nil {
		t.Fatal("new session should have a fresh, uncancelled context")
	}
	if got == sess1 {
		t.Fatal("should be a different session instance")
	}
}

// ---------- Lifecycle: Concurrent Checkin and Ticker ----------

func TestLifecycle_ConcurrentCheckinAndTicker(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize*10),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}

	// Add session with borderline LastCheckin
	sess := newTestSession("concurrent-1",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)
	s.Add(sess)

	var wg sync.WaitGroup
	var panicked atomic.Int32

	// 10 goroutines updating LastCheckin concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicked.Add(1)
				}
			}()
			for j := 0; j < 100; j++ {
				sess.SetLastCheckin(time.Now().Unix())
			}
		}()
	}

	// Simultaneously run ticker callbacks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicked.Add(1)
				}
			}()
			for _, session := range s.All() {
				if !session.isAlived() {
					session.Save()
					s.Remove(session.ID)
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent test did not complete within timeout")
	}

	if p := panicked.Load(); p > 0 {
		t.Fatalf("detected %d panics during concurrent access", p)
	}
}

// ---------- Lifecycle: Multiple sessions with mixed states ----------

func TestLifecycle_MixedAliveDeadSessions(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize*4),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}

	// Add alive sessions
	alive1 := newTestSession("alive-mix-1",
		withExpression("*/5 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)
	alive2 := newTestSession("alive-mix-2",
		withExpression("*/5 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)

	// Add dead sessions
	dead1 := newTestSession("dead-mix-1",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Add(-10*time.Minute).Unix()),
	)
	dead2 := newTestSession("dead-mix-2",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Add(-10*time.Minute).Unix()),
	)

	// Add a bind session (always alive regardless of checkin)
	bind1 := newTestSession("bind-mix-1",
		withType(consts.BindPipeline),
		withLastCheckin(0),
	)

	s.Add(alive1)
	s.Add(alive2)
	s.Add(dead1)
	s.Add(dead2)
	s.Add(bind1)

	// Run ticker
	for _, session := range s.All() {
		if !session.isAlived() {
			session.Save()
			session.Publish(consts.CtrlSessionDead, "dead", true, true)
			s.Remove(session.ID)
		}
	}

	// Alive sessions should remain
	for _, id := range []string{"alive-mix-1", "alive-mix-2", "bind-mix-1"} {
		if _, err := s.Get(id); err != nil {
			t.Fatalf("session %s should still be alive: %v", id, err)
		}
	}

	// Dead sessions should be removed
	for _, id := range []string{"dead-mix-1", "dead-mix-2"} {
		if _, err := s.Get(id); err == nil {
			t.Fatalf("session %s should have been removed", id)
		}
	}

	// Dead session contexts should be cancelled
	if dead1.Ctx.Err() == nil {
		t.Fatal("dead1 context should be cancelled")
	}
	if dead2.Ctx.Err() == nil {
		t.Fatal("dead2 context should be cancelled")
	}

	// Alive/bind session contexts should still be alive
	if alive1.Ctx.Err() != nil {
		t.Fatal("alive1 context should not be cancelled")
	}
	if bind1.Ctx.Err() != nil {
		t.Fatal("bind1 context should not be cancelled")
	}

	remaining := s.All()
	if len(remaining) != 3 {
		t.Fatalf("expected 3 remaining sessions, got %d", len(remaining))
	}
}

// ==========================================================================
// Edge Cases: boundary scenarios and state transition gaps
// ==========================================================================

// ---------- Edge: Register → long silence → ticker kills → Checkin revives ----------

func TestEdge_RegisterLongSilenceThenReborn(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize*4),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}

	// Step 1: Register with short interval
	sess := newTestSession("silence-1",
		withExpression("*/1 * * * *"), // every minute
		withLastCheckin(time.Now().Unix()),
	)
	s.Add(sess)

	// Step 2: NO checkin for a long time — simulate by backdating LastCheckin
	sess.SetLastCheckin(time.Now().Add(-30 * time.Minute).Unix())

	// Step 3: Ticker detects dead
	runTicker(s)

	if _, err := s.Get("silence-1"); err == nil {
		t.Fatal("session should be dead after long silence")
	}
	if sess.Ctx.Err() == nil {
		t.Fatal("context should be cancelled")
	}

	// Collect dead event
	drainEvents(broker, 1)

	// Step 4: Implant comes back — simulate Checkin recovery by adding a new session
	rebornSess := newTestSession("silence-1",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)
	s.Add(rebornSess)

	got, err := s.Get("silence-1")
	if err != nil {
		t.Fatalf("reborn session should be in memory: %v", err)
	}
	if got.Ctx.Err() != nil {
		t.Fatal("reborn session should have fresh context")
	}

	// Step 5: Another ticker run — should NOT kill the fresh session
	runTicker(s)

	if _, err := s.Get("silence-1"); err != nil {
		t.Fatal("fresh session should survive ticker")
	}
}

// ---------- Edge: Task context cascading on session death ----------

func TestEdge_TaskContextCascadesOnSessionDeath(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}
	sess := newTestSession("cascade-1")
	s.Add(sess)

	// Create tasks whose contexts derive from session.Ctx
	task1 := sess.NewTask("type1", 1)
	task2 := sess.NewTask("type2", 5)

	// Tasks should be alive
	if task1.Ctx.Err() != nil {
		t.Fatal("task1 context should be alive before session death")
	}
	if task2.Ctx.Err() != nil {
		t.Fatal("task2 context should be alive before session death")
	}

	// Kill session
	s.Remove("cascade-1")

	// All derived task contexts should be cancelled
	if task1.Ctx.Err() == nil {
		t.Fatal("task1 context should be cancelled after session death")
	}
	if task2.Ctx.Err() == nil {
		t.Fatal("task2 context should be cancelled after session death")
	}
}

// ---------- Edge: Double Remove is safe (idempotent) ----------

func TestEdge_DoubleRemoveIsSafe(t *testing.T) {
	s := &sessions{active: &sync.Map{}}
	sess := newTestSession("double-rm")
	s.Add(sess)

	s.Remove("double-rm")
	if sess.Ctx.Err() == nil {
		t.Fatal("context should be cancelled after first Remove")
	}

	// Second Remove should not panic
	s.Remove("double-rm")

	// Session should still not be accessible
	if _, err := s.Get("double-rm"); err == nil {
		t.Fatal("session should not exist after double Remove")
	}
}

// ---------- Edge: Concurrent Remove (ticker + user delete) ----------

func TestEdge_ConcurrentRemove(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize*10),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	var panicked atomic.Int32
	var wg sync.WaitGroup

	for round := 0; round < 20; round++ {
		s := &sessions{active: &sync.Map{}}
		sess := newTestSession("conc-rm",
			withExpression("*/1 * * * *"),
			withLastCheckin(time.Now().Add(-10*time.Minute).Unix()),
		)
		s.Add(sess)

		wg.Add(2)
		// Ticker path
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicked.Add(1)
				}
			}()
			for _, session := range s.All() {
				if !session.isAlived() {
					session.Save()
					session.Publish(consts.CtrlSessionDead, "dead", true, true)
					s.Remove(session.ID)
				}
			}
		}()
		// User delete path
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicked.Add(1)
				}
			}()
			s.Remove("conc-rm")
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent Remove test timed out")
	}

	if p := panicked.Load(); p > 0 {
		t.Fatalf("detected %d panics during concurrent Remove", p)
	}
}

// ---------- Edge: isAlived with empty Expression ----------

func TestEdge_isAlived_EmptyExpression(t *testing.T) {
	sess := newTestSession("empty-expr",
		withExpression(""), // no expression set
		withLastCheckin(time.Now().Add(-1*time.Hour).Unix()),
	)
	// cronexpr.Parse("") fails → isAlived returns true (never times out)
	// This is a known gap: sessions with no timer expression never die
	if !sess.isAlived() {
		t.Fatal("session with empty expression should be treated as always alive (cronexpr parse error fallback)")
	}
}

// ---------- Edge: isAlived with zero Jitter ----------

func TestEdge_isAlived_ZeroJitter(t *testing.T) {
	sess := newTestSession("zero-jitter",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)
	sess.SessionContext.Jitter = 0.0

	if !sess.isAlived() {
		t.Fatal("recent session with zero jitter should be alive")
	}

	// Now expire it
	sess.SetLastCheckin(time.Now().Add(-10 * time.Minute).Unix())
	if sess.isAlived() {
		t.Fatal("expired session with zero jitter should be dead")
	}
}

// ---------- Edge: isAlived boundary — exactly at allowed offline ----------

func TestEdge_isAlived_Boundary(t *testing.T) {
	// With expression "*/1 * * * *" (every minute):
	// nextInterval ≈ 0..60s, allowedOffline = max(nextInterval*(1+jitter)+30, 150) = 150s
	// So a session that last checked in 149s ago should be alive,
	// and one that checked in 151s ago should be dead.

	sess := newTestSession("boundary",
		withExpression("*/1 * * * *"),
	)
	sess.SessionContext.Jitter = 0.0

	sess.SetLastCheckin(time.Now().Add(-149 * time.Second).Unix())
	if !sess.isAlived() {
		t.Fatal("session 149s ago should be alive (within 150s window)")
	}

	sess.SetLastCheckin(time.Now().Add(-151 * time.Second).Unix())
	if sess.isAlived() {
		t.Fatal("session 151s ago should be dead (beyond 150s window)")
	}
}

// ---------- Edge: Add nil session ----------

func TestEdge_AddNilSession(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}
	result := s.Add(nil)
	if result != nil {
		t.Fatal("Add(nil) should return nil")
	}

	all := s.All()
	if len(all) != 0 {
		t.Fatalf("expected 0 sessions after Add(nil), got %d", len(all))
	}
}

// ---------- Edge: Session Replace (Add with same ID) ----------

func TestEdge_AddReplacesExistingSession(t *testing.T) {
	s := &sessions{active: &sync.Map{}}

	sess1 := newTestSession("replace-1", withLastCheckin(100))
	s.Add(sess1)

	sess2 := newTestSession("replace-1", withLastCheckin(200))
	s.Add(sess2)

	got, _ := s.Get("replace-1")
	if got.LastCheckinUnix() != 200 {
		t.Fatalf("Add should replace: LastCheckin=%d, want 200", got.LastCheckinUnix())
	}

	// Old session's context is NOT cancelled by Add — only Remove cancels
	if sess1.Ctx.Err() != nil {
		t.Fatal("old session context should NOT be cancelled by Add (only by Remove)")
	}
}

// ---------- Edge: Rapid flapping (register → dead → register → dead) ----------

func TestEdge_RapidFlapping(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize*20),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}

	for cycle := 0; cycle < 5; cycle++ {
		// Register
		sess := newTestSession("flap-1",
			withExpression("*/1 * * * *"),
			withLastCheckin(time.Now().Unix()),
		)
		s.Add(sess)

		if _, err := s.Get("flap-1"); err != nil {
			t.Fatalf("cycle %d: session should be alive after register", cycle)
		}

		// Expire it
		sess.SetLastCheckin(time.Now().Add(-10 * time.Minute).Unix())

		// Ticker kills it
		runTicker(s)

		if _, err := s.Get("flap-1"); err == nil {
			t.Fatalf("cycle %d: session should be dead after ticker", cycle)
		}
		if sess.Ctx.Err() == nil {
			t.Fatalf("cycle %d: context should be cancelled", cycle)
		}
	}

	// After 5 flap cycles, no sessions in memory
	if len(s.All()) != 0 {
		t.Fatalf("expected 0 sessions after all flap cycles, got %d", len(s.All()))
	}
}

// ---------- Edge: Recover with mixed finished/unfinished tasks ----------

func TestEdge_RecoverMixedTasks(t *testing.T) {
	sess := newTestSession("recover-mix")

	// 3 finished, 2 unfinished
	for i := uint32(1); i <= 3; i++ {
		task := &Task{Id: i, Cur: 5, Total: 5, SessionId: sess.ID}
		task.Ctx, task.Cancel = context.WithCancel(sess.Ctx)
		sess.Tasks.Add(task)
	}
	for i := uint32(4); i <= 5; i++ {
		task := &Task{Id: i, Cur: 1, Total: 5, SessionId: sess.ID}
		task.Ctx, task.Cancel = context.WithCancel(sess.Ctx)
		sess.Tasks.Add(task)
	}

	sess.Recover()

	// Only unfinished tasks should have response channels
	for i := uint32(1); i <= 3; i++ {
		if _, ok := sess.GetResp(i); ok {
			t.Fatalf("finished task %d should not have response channel", i)
		}
	}
	for i := uint32(4); i <= 5; i++ {
		if _, ok := sess.GetResp(i); !ok {
			t.Fatalf("unfinished task %d should have response channel", i)
		}
	}
}

// ---------- Edge: Session with response channels is cleaned up on death ----------

func TestEdge_ResponseChannelCleanupOnDeath(t *testing.T) {
	s := &sessions{active: &sync.Map{}}
	sess := newTestSession("resp-cleanup")
	s.Add(sess)

	// Store some response channels
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})
	sess.responses.Store(uint32(1), ch1)
	sess.responses.Store(uint32(2), ch2)

	// Verify they exist
	_, ok1 := sess.responses.Load(uint32(1))
	_, ok2 := sess.responses.Load(uint32(2))
	if !ok1 || !ok2 {
		t.Fatal("response channels should exist before death")
	}

	// Remove kills context but does NOT clear responses map
	// (goroutines holding channel refs can still drain)
	s.Remove("resp-cleanup")

	// Context is dead — any goroutine waiting on sess.Ctx should exit
	if sess.Ctx.Err() == nil {
		t.Fatal("context should be cancelled")
	}

	// Response channels still exist in map (not cleaned up by Remove)
	// This is by design — Task.Close() cleans up individual channels
	_, ok1 = sess.responses.Load(uint32(1))
	if !ok1 {
		t.Fatal("response channels should still exist after Remove (cleaned by Task.Close)")
	}
}

// ---------- Edge: Keepalive state across session death ----------

func TestEdge_KeepaliveResetOnDeath(t *testing.T) {
	s := &sessions{active: &sync.Map{}}
	sess := newTestSession("keepalive-death")
	s.Add(sess)

	sess.SetKeepalive(true)
	if !sess.IsKeepaliveEnabled() {
		t.Fatal("keepalive should be enabled")
	}

	s.Remove("keepalive-death")

	if sess.IsKeepaliveEnabled() {
		t.Fatal("keepalive should be reset after Remove")
	}
}

// ---------- Edge: Multiple sessions dying in same ticker cycle ----------

func TestEdge_MultipleDeathsInSameTicker(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	var savedIDs []string
	origSave := sessionDBSave
	sessionDBSave = func(s *models.Session) error {
		savedIDs = append(savedIDs, s.SessionID)
		return nil
	}
	defer func() { sessionDBSave = origSave }()

	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize*10),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	s := &sessions{active: &sync.Map{}}

	// Add 10 dead sessions
	for i := 0; i < 10; i++ {
		sess := newTestSession("batch-dead-"+string(rune('a'+i)),
			withExpression("*/1 * * * *"),
			withLastCheckin(time.Now().Add(-10*time.Minute).Unix()),
		)
		s.Add(sess)
	}

	runTicker(s)

	// All 10 should be removed
	remaining := s.All()
	if len(remaining) != 0 {
		t.Fatalf("expected 0 sessions after batch death, got %d", len(remaining))
	}

	// All 10 should have been saved to DB
	if len(savedIDs) != 10 {
		t.Fatalf("expected 10 DB saves, got %d", len(savedIDs))
	}
}

// ---------- Edge: Checkin saves updated LastCheckin to DB ----------

func TestEdge_CheckinSavePersistsNewTimestamp(t *testing.T) {
	var savedModels []*models.Session
	origSave := sessionDBSave
	sessionDBSave = func(s *models.Session) error {
		// Deep copy to capture snapshot
		cp := *s
		savedModels = append(savedModels, &cp)
		return nil
	}
	defer func() { sessionDBSave = origSave }()

	origArtifact := sessionDBGetArtifact
	origProfile := sessionDBGetProfile
	sessionDBGetArtifact = func(name string) (*models.Artifact, error) {
		return &models.Artifact{Name: name}, nil
	}
	sessionDBGetProfile = func(name string) (*models.Profile, error) {
		return &models.Profile{Name: name}, nil
	}
	defer func() {
		sessionDBGetArtifact = origArtifact
		sessionDBGetProfile = origProfile
	}()

	sess := newTestSession("checkin-ts",
		withLastCheckin(1000),
	)

	// Simulate checkin: update LastCheckin then Save
	newTime := time.Now().Unix()
	sess.SetLastCheckin(newTime)
	sess.Save()

	if len(savedModels) == 0 {
		t.Fatal("expected at least one save")
	}
	if savedModels[len(savedModels)-1].LastCheckin != newTime {
		t.Fatalf("DB should have new LastCheckin=%d, got %d",
			newTime, savedModels[len(savedModels)-1].LastCheckin)
	}
}

// ==========================================================================
// Helper: simulate ticker callback
// ==========================================================================

func runTicker(s *sessions) {
	for _, session := range s.All() {
		if !session.isAlived() {
			session.Save()
			session.Publish(consts.CtrlSessionDead,
				"session dead", true, true)
			s.Remove(session.ID)
		}
	}
}

func drainEvents(broker *eventBroker, count int) {
	for i := 0; i < count; i++ {
		select {
		case <-broker.publish:
		case <-time.After(500 * time.Millisecond):
			return
		}
	}
}

// ---------- Lifecycle: Save on checkin persists LastCheckin ----------

func TestLifecycle_SaveOnCheckin(t *testing.T) {
	var savedModels []*models.Session
	origSave := sessionDBSave
	sessionDBSave = func(s *models.Session) error {
		savedModels = append(savedModels, s)
		return nil
	}
	defer func() { sessionDBSave = origSave }()

	origArtifact := sessionDBGetArtifact
	origProfile := sessionDBGetProfile
	sessionDBGetArtifact = func(name string) (*models.Artifact, error) {
		return &models.Artifact{Name: name}, nil
	}
	sessionDBGetProfile = func(name string) (*models.Profile, error) {
		return &models.Profile{Name: name}, nil
	}
	defer func() {
		sessionDBGetArtifact = origArtifact
		sessionDBGetProfile = origProfile
	}()

	sess := newTestSession("checkin-save")
	checkinTime := time.Now().Unix()
	sess.SetLastCheckin(checkinTime)
	sess.Save()

	if len(savedModels) == 0 {
		t.Fatal("Save should have been called")
	}

	lastSaved := savedModels[len(savedModels)-1]
	if lastSaved.LastCheckin != checkinTime {
		t.Fatalf("saved LastCheckin = %d, want %d", lastSaved.LastCheckin, checkinTime)
	}
}
