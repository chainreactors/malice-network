package core

import (
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
	sess.LastCheckin = time.Now().Unix()
	sess.Save()

	// Step 3: Expire — set old LastCheckin
	sess.LastCheckin = time.Now().Add(-10 * time.Minute).Unix()

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
				sess.LastCheckin = time.Now().Unix()
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
	sess.LastCheckin = checkinTime
	sess.Save()

	if len(savedModels) == 0 {
		t.Fatal("Save should have been called")
	}

	lastSaved := savedModels[len(savedModels)-1]
	if lastSaved.LastCheckin != checkinTime {
		t.Fatalf("saved LastCheckin = %d, want %d", lastSaved.LastCheckin, checkinTime)
	}
}
