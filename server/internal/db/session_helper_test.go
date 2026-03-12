package db

import (
	"testing"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// ============================================
// Operator Tests
// ============================================

func TestCreateAndFindOperator(t *testing.T) {
	initTestDB(t)

	op := &models.Operator{
		Name:        "op-test-1",
		Type:        "client",
		Fingerprint: "fp-abc-123",
	}
	if err := CreateOperator(op); err != nil {
		t.Fatalf("CreateOperator failed: %v", err)
	}

	// FindOperatorByName
	found, err := FindOperatorByName("op-test-1")
	if err != nil {
		t.Fatalf("FindOperatorByName failed: %v", err)
	}
	if found.Name != "op-test-1" {
		t.Errorf("expected name 'op-test-1', got %q", found.Name)
	}

	// FindOperatorByFingerprint
	found2, err := FindOperatorByFingerprint("fp-abc-123")
	if err != nil {
		t.Fatalf("FindOperatorByFingerprint failed: %v", err)
	}
	if found2.Name != "op-test-1" {
		t.Errorf("expected name 'op-test-1', got %q", found2.Name)
	}
}

func TestRemoveOperator(t *testing.T) {
	initTestDB(t)

	op := &models.Operator{Name: "op-rm-1", Type: "client", Fingerprint: "fp-rm-1"}
	CreateOperator(op)

	if err := RemoveOperator("op-rm-1"); err != nil {
		t.Fatalf("RemoveOperator failed: %v", err)
	}

	_, err := FindOperatorByName("op-rm-1")
	if err == nil {
		t.Error("expected error after removing operator, got nil")
	}
}

func TestHasOperator(t *testing.T) {
	initTestDB(t)

	has, err := HasOperator("client")
	if err != nil {
		t.Fatalf("HasOperator failed: %v", err)
	}
	if has {
		t.Error("expected no operators initially")
	}

	CreateOperator(&models.Operator{Name: "op-has-1", Type: "client", Fingerprint: "fp-has-1"})

	has, err = HasOperator("client")
	if err != nil {
		t.Fatalf("HasOperator failed: %v", err)
	}
	if !has {
		t.Error("expected HasOperator to return true after creating one")
	}
}

func TestListClientsAndListeners(t *testing.T) {
	initTestDB(t)

	CreateOperator(&models.Operator{Name: "cl-1", Type: "client", Fingerprint: "fp-cl-1"})
	CreateOperator(&models.Operator{Name: "ls-1", Type: "listener", Fingerprint: "fp-ls-1"})

	clients, err := ListClients()
	if err != nil {
		t.Fatalf("ListClients failed: %v", err)
	}
	if len(clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(clients))
	}

	listeners, err := ListListeners()
	if err != nil {
		t.Fatalf("ListListeners failed: %v", err)
	}
	if len(listeners) != 1 {
		t.Errorf("expected 1 listener, got %d", len(listeners))
	}
}

func TestRevokeOperator(t *testing.T) {
	initTestDB(t)

	CreateOperator(&models.Operator{Name: "op-revoke", Type: "client", Fingerprint: "fp-revoke"})

	if err := RevokeOperator("op-revoke"); err != nil {
		t.Fatalf("RevokeOperator failed: %v", err)
	}

	found, _ := FindOperatorByName("op-revoke")
	if !found.Revoked {
		t.Error("expected operator to be revoked")
	}
}

// ============================================
// Session CRUD Tests
// ============================================

func TestCreateOrRecoverSession(t *testing.T) {
	initTestDB(t)

	sess := &models.Session{
		SessionID: "cor-sess-1",
		GroupName: "test",
		Data:      &client.SessionContext{},
	}
	if err := CreateOrRecoverSession(sess); err != nil {
		t.Fatalf("CreateOrRecoverSession failed: %v", err)
	}

	found, err := FindSession("cor-sess-1")
	if err != nil {
		t.Fatalf("FindSession failed: %v", err)
	}
	if found.SessionID != "cor-sess-1" {
		t.Errorf("expected 'cor-sess-1', got %q", found.SessionID)
	}
}

func TestCreateOrRecoverSession_Recover(t *testing.T) {
	initTestDB(t)

	// Create initial session
	sess := &models.Session{
		SessionID: "cor-recover-1",
		GroupName: "old-group",
		Data:      &client.SessionContext{},
	}
	CreateOrRecoverSession(sess)

	// Create again with same ID (should delete and recreate)
	sess2 := &models.Session{
		SessionID: "cor-recover-1",
		GroupName: "new-group",
		Data:      &client.SessionContext{},
	}
	if err := CreateOrRecoverSession(sess2); err != nil {
		t.Fatalf("CreateOrRecoverSession (recover) failed: %v", err)
	}

	found, _ := FindSession("cor-recover-1")
	if found.GroupName != "new-group" {
		t.Errorf("expected 'new-group', got %q", found.GroupName)
	}
}

func TestFindSession_Removed(t *testing.T) {
	initTestDB(t)

	sess := &models.Session{
		SessionID: "find-removed-1",
		IsRemoved: true,
		Data:      &client.SessionContext{},
	}
	Session().Create(sess)

	found, err := FindSession("find-removed-1")
	if err != nil {
		t.Fatalf("FindSession should not error for removed sessions, got: %v", err)
	}
	if found != nil {
		t.Error("FindSession should return nil for removed sessions")
	}
}

func TestRemoveSession(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "rm-sess-1", Data: &client.SessionContext{}})

	if err := RemoveSession("rm-sess-1"); err != nil {
		t.Fatalf("RemoveSession failed: %v", err)
	}

	found, _ := FindSession("rm-sess-1")
	if found != nil {
		t.Error("expected session to be removed")
	}
}

func TestRecoverRemovedSession(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{
		SessionID: "recover-1",
		IsRemoved: true,
		Data:      &client.SessionContext{},
	})

	recovered, err := RecoverRemovedSession("recover-1")
	if err != nil {
		t.Fatalf("RecoverRemovedSession failed: %v", err)
	}
	if recovered.IsRemoved {
		t.Error("expected session to no longer be removed")
	}

	// Should now be findable
	found, _ := FindSession("recover-1")
	if found == nil {
		t.Error("expected to find recovered session")
	}
}

func TestUpdateSession(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{
		SessionID: "upd-sess-1",
		GroupName: "old",
		Note:      "old-note",
		Data:      &client.SessionContext{},
	})

	if err := UpdateSession("upd-sess-1", "new-note", "new-group"); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}

	found, _ := FindSession("upd-sess-1")
	if found.Note != "new-note" {
		t.Errorf("expected note 'new-note', got %q", found.Note)
	}
	if found.GroupName != "new-group" {
		t.Errorf("expected group 'new-group', got %q", found.GroupName)
	}
}

func TestUpdateSessionTimer(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{
		SessionID: "timer-sess-1",
		Data: &client.SessionContext{
			SessionInfo: &client.SessionInfo{Expression: "0 */5 * * *"},
		},
	})

	if err := UpdateSessionTimer("timer-sess-1", "0 */10 * * *", 0.5); err != nil {
		t.Fatalf("UpdateSessionTimer failed: %v", err)
	}

	var result models.Session
	Session().Where("session_id = ?", "timer-sess-1").First(&result)
	if result.Data.Expression != "0 */10 * * *" {
		t.Errorf("expected expression '0 */10 * * *', got %q", result.Data.Expression)
	}
	if result.Data.Jitter != 0.5 {
		t.Errorf("expected jitter 0.5, got %f", result.Data.Jitter)
	}
}

func TestSaveSessionModel(t *testing.T) {
	initTestDB(t)

	sess := &models.Session{
		SessionID: "save-model-1",
		GroupName: "initial",
		Data:      &client.SessionContext{},
	}
	Session().Create(sess)

	sess.GroupName = "modified"
	if err := SaveSessionModel(sess); err != nil {
		t.Fatalf("SaveSessionModel failed: %v", err)
	}

	var result models.Session
	Session().Where("session_id = ?", "save-model-1").First(&result)
	if result.GroupName != "modified" {
		t.Errorf("expected 'modified', got %q", result.GroupName)
	}
}

// ============================================
// Task CRUD Tests
// ============================================

func TestAddAndGetTask(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "task-sess-1"})

	task := &clientpb.Task{
		SessionId: "task-sess-1",
		TaskId:    1,
		Type:      "exec",
		Total:     100,
	}
	if err := AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	found, err := GetTask("task-sess-1-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if found.Type != "exec" {
		t.Errorf("expected type 'exec', got %q", found.Type)
	}
	if found.Total != 100 {
		t.Errorf("expected total 100, got %d", found.Total)
	}
}

func TestGetTaskBySessionAndSeq(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "task-seq-sess"})
	AddTask(&clientpb.Task{SessionId: "task-seq-sess", TaskId: 5, Type: "download"})

	found, err := GetTaskBySessionAndSeq("task-seq-sess", 5)
	if err != nil {
		t.Fatalf("GetTaskBySessionAndSeq failed: %v", err)
	}
	if found.Seq != 5 {
		t.Errorf("expected seq 5, got %d", found.Seq)
	}
}

func TestUpdateTaskFinish(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "task-fin-sess"})
	AddTask(&clientpb.Task{SessionId: "task-fin-sess", TaskId: 1, Type: "exec", Total: 10})

	if err := UpdateTaskFinish("task-fin-sess-1"); err != nil {
		t.Fatalf("UpdateTaskFinish failed: %v", err)
	}

	found, _ := GetTask("task-fin-sess-1")
	if found.FinishTime.IsZero() {
		t.Error("expected FinishTime to be set after UpdateTaskFinish")
	}
}

func TestUpdateTaskDescription(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "task-desc-sess"})
	AddTask(&clientpb.Task{SessionId: "task-desc-sess", TaskId: 1, Type: "exec"})

	if err := UpdateTaskDescription("task-desc-sess-1", "test description"); err != nil {
		t.Fatalf("UpdateTaskDescription failed: %v", err)
	}

	found, _ := GetTask("task-desc-sess-1")
	if found.Description != "test description" {
		t.Errorf("expected 'test description', got %q", found.Description)
	}
}

func TestFindTaskAndMaxTasksID(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "task-max-sess"})
	AddTask(&clientpb.Task{SessionId: "task-max-sess", TaskId: 3, Type: "exec"})
	AddTask(&clientpb.Task{SessionId: "task-max-sess", TaskId: 7, Type: "upload"})
	AddTask(&clientpb.Task{SessionId: "task-max-sess", TaskId: 5, Type: "download"})

	tasks, maxID, err := FindTaskAndMaxTasksID("task-max-sess")
	if err != nil {
		t.Fatalf("FindTaskAndMaxTasksID failed: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
	if maxID != 7 {
		t.Errorf("expected max ID 7, got %d", maxID)
	}
}
