package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// newTestSession creates a minimal Session for testing without DB/filesystem dependencies.
func newTestSession(id string, opts ...func(*Session)) *Session {
	sess := &Session{
		ID:     id,
		Type:   "test",
		Name:   "test-session",
		Group:  "default",
		Target: "127.0.0.1",
		Tasks:  NewTasks(),
		SessionContext: &client.SessionContext{
			SessionInfo: &client.SessionInfo{
				Expression: "*/5 * * * *",
				Jitter:     0.1,
			},
		},
		responses:   &sync.Map{},
	}
	sess.Ctx, sess.Cancel = context.WithCancel(context.Background())
	sess.SetLastCheckin(time.Now().Unix())
	for _, opt := range opts {
		opt(sess)
	}
	return sess
}

// withType returns an option that sets the session type.
func withType(t string) func(*Session) {
	return func(s *Session) { s.Type = t }
}

// withLastCheckin returns an option that sets the last checkin timestamp.
func withLastCheckin(ts int64) func(*Session) {
	return func(s *Session) { s.SetLastCheckin(ts) }
}

// withExpression returns an option that sets the cron expression.
func withExpression(expr string) func(*Session) {
	return func(s *Session) { s.SessionContext.Expression = expr }
}

// installTestDBMocks replaces DB function variables with no-op mocks.
// Returns a cleanup function to restore originals.
func installTestDBMocks() func() {
	origSave := sessionDBSave
	origArtifact := sessionDBGetArtifact
	origProfile := sessionDBGetProfile

	sessionDBSave = func(s *models.Session) error { return nil }
	sessionDBGetArtifact = func(name string) (*models.Artifact, error) {
		return &models.Artifact{Name: name}, nil
	}
	sessionDBGetProfile = func(name string) (*models.Profile, error) {
		return &models.Profile{Name: name}, nil
	}

	return func() {
		sessionDBSave = origSave
		sessionDBGetArtifact = origArtifact
		sessionDBGetProfile = origProfile
	}
}

// ---------- sessions CRUD ----------

func TestSessions_AddGetRemove(t *testing.T) {
	s := &sessions{active: &sync.Map{}}

	sess := newTestSession("sess-1")
	s.Add(sess)

	got, err := s.Get("sess-1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != "sess-1" {
		t.Fatalf("expected ID sess-1, got %s", got.ID)
	}

	s.Remove("sess-1")

	_, err = s.Get("sess-1")
	if err == nil {
		t.Fatal("expected error after Remove, got nil")
	}
}

func TestSessions_Remove_CancelsContext(t *testing.T) {
	s := &sessions{active: &sync.Map{}}
	sess := newTestSession("sess-cancel")
	s.Add(sess)

	// Context should be alive before Remove
	if sess.Ctx.Err() != nil {
		t.Fatal("context should not be cancelled before Remove")
	}

	s.Remove("sess-cancel")

	// Context should be cancelled after Remove
	if sess.Ctx.Err() == nil {
		t.Fatal("context should be cancelled after Remove")
	}
}

func TestSessions_Remove_NonExistent(t *testing.T) {
	s := &sessions{active: &sync.Map{}}
	// Should not panic
	s.Remove("does-not-exist")
}

func TestSessions_All_Snapshot(t *testing.T) {
	s := &sessions{active: &sync.Map{}}

	for i := 0; i < 3; i++ {
		sess := newTestSession("snap-" + string(rune('a'+i)))
		s.Add(sess)
	}

	all := s.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(all))
	}

	// Modifying the slice should not affect internal map
	all = all[:0]
	all2 := s.All()
	if len(all2) != 3 {
		t.Fatalf("expected 3 sessions after slice modification, got %d", len(all2))
	}
}

// ---------- isAlived ----------

func TestSession_isAlived_BindPipeline(t *testing.T) {
	sess := newTestSession("bind-1",
		withType(consts.BindPipeline),
		withLastCheckin(0), // very old, but bind is always alive
	)
	if !sess.isAlived() {
		t.Fatal("BindPipeline session should always be alive")
	}
}

func TestSession_isAlived_Expired(t *testing.T) {
	sess := newTestSession("expired-1",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Add(-10*time.Minute).Unix()),
	)
	if sess.isAlived() {
		t.Fatal("session with old LastCheckin should be dead")
	}
}

func TestSession_isAlived_Recent(t *testing.T) {
	sess := newTestSession("alive-1",
		withExpression("*/5 * * * *"),
		withLastCheckin(time.Now().Unix()),
	)
	if !sess.isAlived() {
		t.Fatal("session with recent LastCheckin should be alive")
	}
}

func TestSession_isAlived_NilSession(t *testing.T) {
	var sess *Session
	if sess.isAlived() {
		t.Fatal("nil session should not be alive")
	}
}

// ---------- Recover ----------

func TestSession_Recover_UsesCorrectKey(t *testing.T) {
	sess := newTestSession("recover-1")

	// Add some tasks with known IDs
	unfinishedTask := &Task{
		Id:        10,
		Type:      "test",
		SessionId: "recover-1",
		Cur:       0,
		Total:     3,
	}
	unfinishedTask.Ctx, unfinishedTask.Cancel = context.WithCancel(sess.Ctx)
	sess.Tasks.Add(unfinishedTask)

	finishedTask := &Task{
		Id:        20,
		Type:      "test",
		SessionId: "recover-1",
		Cur:       5,
		Total:     5,
	}
	finishedTask.Ctx, finishedTask.Cancel = context.WithCancel(sess.Ctx)
	sess.Tasks.Add(finishedTask)

	err := sess.Recover()
	if err != nil {
		t.Fatalf("Recover failed: %v", err)
	}

	// Unfinished task should have a response channel accessible by uint32 ID
	ch, ok := sess.GetResp(10)
	if !ok {
		t.Fatal("expected response channel for unfinished task 10")
	}
	if ch == nil {
		t.Fatal("response channel should not be nil")
	}

	// Finished task should NOT have a response channel
	_, ok = sess.GetResp(20)
	if ok {
		t.Fatal("finished task 20 should not have a response channel")
	}
}

// ---------- Task management ----------

func TestSession_NewTask_IncrementsSeq(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	sess := newTestSession("task-seq")

	task1 := sess.NewTask("type1", 1)
	task2 := sess.NewTask("type2", 1)

	if task2.Id <= task1.Id {
		t.Fatalf("task IDs should increment: task1=%d, task2=%d", task1.Id, task2.Id)
	}
}

func TestSession_StoreGetRemoveResp(t *testing.T) {
	sess := newTestSession("resp-1")

	ch := make(chan *implantpb.Spite, 16)
	sess.StoreResp(42, ch)

	got, ok := sess.GetResp(42)
	if !ok {
		t.Fatal("expected to find response channel for task 42")
	}
	if got != ch {
		t.Fatal("returned channel should be the same as stored")
	}

	// RemoveResp removes without closing
	sess.RemoveResp(42)
	_, ok = sess.GetResp(42)
	if ok {
		t.Fatal("response channel should be removed after RemoveResp")
	}

	// Channel should still be open (not closed)
	select {
	case ch <- &implantpb.Spite{}:
		// good, can still send
	default:
		t.Fatal("channel should still be open after RemoveResp")
	}
}

func TestSession_DeleteResp_ClosesChannel(t *testing.T) {
	sess := newTestSession("resp-2")

	ch := make(chan *implantpb.Spite, 16)
	sess.StoreResp(99, ch)

	sess.DeleteResp(99)

	_, ok := sess.GetResp(99)
	if ok {
		t.Fatal("response channel should be deleted after DeleteResp")
	}

	// Channel should be closed
	_, open := <-ch
	if open {
		t.Fatal("channel should be closed after DeleteResp")
	}
}

// ---------- Save with mocked DB ----------

func TestSession_Save_UsesMockedDB(t *testing.T) {
	var savedModel *models.Session
	origSave := sessionDBSave
	sessionDBSave = func(s *models.Session) error {
		savedModel = s
		return nil
	}
	defer func() { sessionDBSave = origSave }()

	// Also mock artifact/profile lookups used by ToModel
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

	sess := newTestSession("save-1")
	err := sess.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if savedModel == nil {
		t.Fatal("Save should have called sessionDBSave")
	}
	if savedModel.SessionID != "save-1" {
		t.Fatalf("saved session ID = %s, want save-1", savedModel.SessionID)
	}
}

// ---------- Keepalive ----------

func TestSession_Keepalive(t *testing.T) {
	sess := newTestSession("keepalive-1")

	if sess.IsKeepaliveEnabled() {
		t.Fatal("keepalive should be disabled by default")
	}

	prev := sess.SetKeepalive(true)
	if prev {
		t.Fatal("previous keepalive state should be false")
	}
	if !sess.IsKeepaliveEnabled() {
		t.Fatal("keepalive should be enabled after SetKeepalive(true)")
	}

	sess.ResetKeepalive()
	if sess.IsKeepaliveEnabled() {
		t.Fatal("keepalive should be disabled after Reset")
	}
}

func TestSessions_SweepInactiveKeepsPendingTasks(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	s := &sessions{active: &sync.Map{}}
	sess := newTestSession("sweep-pending",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Add(-10*time.Minute).Unix()),
	)
	task := sess.NewTask("sleep", 1)
	s.Add(sess)

	s.SweepInactive()

	if _, err := s.Get("sweep-pending"); err != nil {
		t.Fatalf("session with unfinished tasks should remain in memory: %v", err)
	}
	if !sess.IsMarkedDead() {
		t.Fatal("session with unfinished tasks should still be marked dead")
	}
	if sess.Ctx.Err() != nil {
		t.Fatal("session context should stay alive while unfinished tasks exist")
	}
	if task.Ctx.Err() != nil {
		t.Fatal("unfinished task context should stay alive while session is retained")
	}
}

func TestSessions_SweepInactiveRemovesIdleSessions(t *testing.T) {
	cleanup := installTestDBMocks()
	defer cleanup()

	s := &sessions{active: &sync.Map{}}
	sess := newTestSession("sweep-idle",
		withExpression("*/1 * * * *"),
		withLastCheckin(time.Now().Add(-10*time.Minute).Unix()),
	)
	s.Add(sess)

	s.SweepInactive()

	if _, err := s.Get("sweep-idle"); err == nil {
		t.Fatal("idle dead session should be removed from memory")
	}
	if !sess.IsMarkedDead() {
		t.Fatal("removed idle session should have been marked dead first")
	}
	if sess.Ctx.Err() == nil {
		t.Fatal("removed idle session context should be cancelled")
	}
}

// ---------- Helper test: newTestBroker already defined in safe_test.go ----------
// We rely on the newTestBroker() from safe_test.go since we're in the same package.
// No need to redefine it.

func init() {
	// Ensure EventBroker is not nil for tests that need it.
	// safe_test.go tests replace it per-test; we just need a default.
	if EventBroker == nil {
		EventBroker = newTestBroker()
		GoGuarded("test-event-broker", func() error {
			EventBroker.Start()
			return nil
		}, LogGuardedError("test-event-broker"))
	}
}
