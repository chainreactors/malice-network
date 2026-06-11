package rpc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/protobuf/proto"
)

// ---------------------------------------------------------------------------
// GetTasks
// ---------------------------------------------------------------------------

func TestGetTasks_ActiveSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gt-active-sess", "gt-active-pipe", true)

	// Seed a task in the DB for this session.
	taskPb := &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    1,
		Type:      "test",
	}
	if err := db.AddTask(taskPb); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Request with All=true to go through DB path.
	resp, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: sess.ID,
		All:       true,
	})
	if err != nil {
		t.Fatalf("GetTasks(All=true) error: %v", err)
	}
	if resp == nil {
		t.Fatal("GetTasks returned nil")
	}
}

func TestGetTasks_ActiveOnly(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gt-mem-sess", "gt-mem-pipe", true)

	// Create an in-memory task via the session.
	sess.NewTask("test-task", 1)

	resp, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: sess.ID,
		All:       false,
	})
	if err != nil {
		t.Fatalf("GetTasks(All=false) error: %v", err)
	}
	if resp == nil || len(resp.Tasks) == 0 {
		t.Fatal("expected at least one in-memory task")
	}
}

func TestGetTasks_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).GetTasks(context.Background(), nil)
	if !errors.Is(err, types.ErrMissingSessionRequestField) {
		t.Fatalf("GetTasks(nil) error = %v, want %v", err, types.ErrMissingSessionRequestField)
	}
}

func TestGetTasks_EmptySessionID(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{})
	if !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("GetTasks(empty SessionId) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
}

func TestGetTasks_MissingSession(t *testing.T) {
	_ = newRPCTestEnv(t)
	// Session not in memory, DB fallback returns empty task list (no error).
	resp, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: "nonexistent",
		All:       false,
	})
	if err != nil {
		t.Fatalf("GetTasks(missing session) error = %v, want nil (DB fallback)", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response from DB fallback")
	}
	if len(resp.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(resp.Tasks))
	}
}

func TestGetTasks_EmptyResult(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gt-empty-sess", "gt-empty-pipe", true)

	resp, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: sess.ID,
		All:       true,
	})
	if err != nil {
		t.Fatalf("GetTasks error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response for session with no tasks")
	}
	if len(resp.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(resp.Tasks))
	}
}

// ---------------------------------------------------------------------------
// GetTaskContent
// ---------------------------------------------------------------------------

func TestGetTaskContent_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).GetTaskContent(context.Background(), nil)
	if !errors.Is(err, types.ErrMissingSessionRequestField) {
		t.Fatalf("GetTaskContent(nil) error = %v, want %v", err, types.ErrMissingSessionRequestField)
	}
}

func TestGetTaskContent_EmptySessionID(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).GetTaskContent(context.Background(), &clientpb.Task{})
	if !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("GetTaskContent(empty session) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
}

func TestGetTaskContent_MissingSession(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).GetTaskContent(context.Background(), &clientpb.Task{
		SessionId: "nonexistent",
		TaskId:    1,
	})
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("GetTaskContent(missing session) error = %v, want %v", err, types.ErrNotFoundSession)
	}
}

func TestGetTaskContent_MissingTask(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gtc-miss-task", "gtc-miss-pipe", true)

	_, err := (&Server{}).GetTaskContent(context.Background(), &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    9999,
	})
	if !errors.Is(err, types.ErrNotFoundTask) {
		t.Fatalf("GetTaskContent(missing task) error = %v, want %v", err, types.ErrNotFoundTask)
	}
}

// ---------------------------------------------------------------------------
// ListTasks (AssertAndHandle wrapper)
// ---------------------------------------------------------------------------

func TestListTasks_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).ListTasks(context.Background(), nil)
	if !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("ListTasks(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestListTasks_WrongModuleName(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "lt-wrong-mod", "lt-wrong-pipe", true)

	_, err := (&Server{}).ListTasks(incomingSessionContext(sess.ID), &implantpb.Request{
		Name: "wrong_module",
	})
	if !errors.Is(err, types.ErrAssertFailure) {
		t.Fatalf("ListTasks(wrong module) error = %v, want %v", err, types.ErrAssertFailure)
	}
}

func TestListTasks_NoSessionInContext(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).ListTasks(context.Background(), &implantpb.Request{
		Name: consts.ModuleListTask,
	})
	// AssertAndHandle delegates to GenericInternal which calls getSession.
	// No session_id in context means ErrNotFoundSession.
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("ListTasks(no session ctx) error = %v, want %v", err, types.ErrNotFoundSession)
	}
}

// ---------------------------------------------------------------------------
// CancelTask
// ---------------------------------------------------------------------------

func TestCancelTask_NoSessionInContext(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).CancelTask(context.Background(), &implantpb.TaskCtrl{TaskId: 1})
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("CancelTask(no ctx) error = %v, want %v", err, types.ErrNotFoundSession)
	}
}

func TestCancelTask_MissingTask(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "ct-miss-task", "ct-miss-pipe", true)

	_, err := (&Server{}).CancelTask(incomingSessionContext(sess.ID), &implantpb.TaskCtrl{TaskId: 9999})
	if !errors.Is(err, types.ErrNotFoundTask) {
		t.Fatalf("CancelTask(missing task) error = %v, want %v", err, types.ErrNotFoundTask)
	}
}

func TestCancelTask_NoPipeline(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "ct-nopipe", "ct-nopipe-pipe", true)

	// Create a task in the session.
	task := sess.NewTask("cancel-target", 1)

	// Make sure there is no pipeline stream registered.
	pipelinesCh.Delete(sess.PipelineID)

	_, err := (&Server{}).CancelTask(incomingSessionContext(sess.ID), &implantpb.TaskCtrl{TaskId: task.Id})
	if !errors.Is(err, types.ErrNotFoundPipeline) {
		t.Fatalf("CancelTask(no pipeline) error = %v, want %v", err, types.ErrNotFoundPipeline)
	}
}

// ---------------------------------------------------------------------------
// WaitTaskContent
// ---------------------------------------------------------------------------

func TestWaitTaskContent_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).WaitTaskContent(context.Background(), nil)
	if !errors.Is(err, types.ErrMissingSessionRequestField) {
		t.Fatalf("WaitTaskContent(nil) error = %v, want %v", err, types.ErrMissingSessionRequestField)
	}
}

func TestWaitTaskContent_EmptySessionID(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).WaitTaskContent(context.Background(), &clientpb.Task{})
	if !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("WaitTaskContent(empty sid) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
}

// ---------------------------------------------------------------------------
// WaitTaskFinish
// ---------------------------------------------------------------------------

func TestWaitTaskFinish_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).WaitTaskFinish(context.Background(), nil)
	if !errors.Is(err, types.ErrMissingSessionRequestField) {
		t.Fatalf("WaitTaskFinish(nil) error = %v, want %v", err, types.ErrMissingSessionRequestField)
	}
}

// ---------------------------------------------------------------------------
// GetAllTaskContent
// ---------------------------------------------------------------------------

func TestGetAllTaskContent_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).GetAllTaskContent(context.Background(), nil)
	if !errors.Is(err, types.ErrMissingSessionRequestField) {
		t.Fatalf("GetAllTaskContent(nil) error = %v, want %v", err, types.ErrMissingSessionRequestField)
	}
}

func TestGetAllTaskContent_MissingSession(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).GetAllTaskContent(context.Background(), &clientpb.Task{
		SessionId: "no-such-session",
		TaskId:    1,
	})
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("GetAllTaskContent(missing session) error = %v, want %v", err, types.ErrNotFoundSession)
	}
}

func TestGetAllTaskContent_MissingTask(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gatc-miss", "gatc-pipe", true)

	_, err := (&Server{}).GetAllTaskContent(context.Background(), &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    9999,
	})
	if !errors.Is(err, types.ErrNotFoundTask) {
		t.Fatalf("GetAllTaskContent(missing task) error = %v, want %v", err, types.ErrNotFoundTask)
	}
}

// ---------------------------------------------------------------------------
// QueryTask - no session in context
// ---------------------------------------------------------------------------

func TestQueryTask_NoSessionInContext(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).QueryTask(context.Background(), &implantpb.TaskCtrl{TaskId: 1})
	// newGenericRequest calls getSession which requires session_id in metadata.
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("QueryTask(no ctx) error = %v, want %v", err, types.ErrNotFoundSession)
	}
}

// ---------------------------------------------------------------------------
// Concurrency: GetTasks on session being removed
// ---------------------------------------------------------------------------

func TestGetTasks_SessionRemovedMidFlight(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gt-race-sess", "gt-race-pipe", true)

	// Remove session from memory.
	core.Sessions.Remove(sess.ID)

	// Request active-only tasks should now fallback to DB (no error, empty list).
	resp, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: sess.ID,
		All:       false,
	})
	if err != nil {
		t.Fatalf("GetTasks after removal error = %v, want nil (DB fallback)", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response from DB fallback")
	}

	// All=true still works (DB path).
	resp, err = (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: sess.ID,
		All:       true,
	})
	if err != nil {
		t.Fatalf("GetTasks(All=true) after removal error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response from DB path")
	}
}

// ---------------------------------------------------------------------------
// DB Fallback: helpers
// ---------------------------------------------------------------------------

// writeTaskSpiteToDisk creates a task spite file on disk for testing DB fallback paths.
func writeTaskSpiteToDisk(t testing.TB, sessionID string, taskSeq uint32, index int, spite *implantpb.Spite) {
	t.Helper()
	taskDir := filepath.Join(configs.ContextPath, sessionID, consts.TaskPath)
	if err := os.MkdirAll(taskDir, 0o700); err != nil {
		t.Fatalf("MkdirAll task dir: %v", err)
	}
	data, err := proto.Marshal(spite)
	if err != nil {
		t.Fatalf("Marshal spite: %v", err)
	}
	fileName := fmt.Sprintf("%d_%d", taskSeq, index)
	if err := os.WriteFile(filepath.Join(taskDir, fileName), data, 0o600); err != nil {
		t.Fatalf("WriteFile task spite: %v", err)
	}
}

// seedDBOnlyTask creates a session (saved to DB, removed from memory) with a task
// and disk-persisted spite, returning the session ID and task seq.
func seedDBOnlyTask(t testing.TB, env *rpcTestEnv, prefix string) (sessionID string, taskSeq uint32) {
	t.Helper()
	sessID := prefix + "-sess"
	pipeID := prefix + "-pipe"
	sess := env.seedSession(t, sessID, pipeID, true)

	taskSeq = uint32(1)
	taskPb := &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    taskSeq,
		Type:      "test-db-fallback",
		Cur:       1,
		Total:     1,
	}
	if err := db.AddTask(taskPb); err != nil {
		t.Fatalf("AddTask: %v", err)
	}

	writeTaskSpiteToDisk(t, sess.ID, taskSeq, 0, &implantpb.Spite{
		TaskId: taskSeq,
		Status: &implantpb.Status{TaskId: taskSeq},
	})

	// Remove session from memory to simulate dead session.
	core.Sessions.Remove(sess.ID)
	return sess.ID, taskSeq
}

// ---------------------------------------------------------------------------
// DB Fallback: GetTaskContent
// ---------------------------------------------------------------------------

func TestGetTaskContent_DBFallback(t *testing.T) {
	env := newRPCTestEnv(t)
	sessID, taskSeq := seedDBOnlyTask(t, env, "gtc-dbfb")

	resp, err := (&Server{}).GetTaskContent(context.Background(), &clientpb.Task{
		SessionId: sessID,
		TaskId:    taskSeq,
		Need:      -1,
	})
	if err != nil {
		t.Fatalf("GetTaskContent DB fallback error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Task == nil || resp.Task.TaskId != taskSeq {
		t.Fatalf("expected task %d, got %v", taskSeq, resp.Task)
	}
	if resp.Session == nil || resp.Session.SessionId != sessID {
		t.Fatalf("expected session %s, got %v", sessID, resp.Session)
	}
	if resp.Spite == nil {
		t.Fatal("expected non-nil spite")
	}
}

// ---------------------------------------------------------------------------
// DB Fallback: GetAllTaskContent
// ---------------------------------------------------------------------------

func TestGetAllTaskContent_DBFallback(t *testing.T) {
	env := newRPCTestEnv(t)
	sessID, taskSeq := seedDBOnlyTask(t, env, "gatc-dbfb")

	resp, err := (&Server{}).GetAllTaskContent(context.Background(), &clientpb.Task{
		SessionId: sessID,
		TaskId:    taskSeq,
	})
	if err != nil {
		t.Fatalf("GetAllTaskContent DB fallback error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Spites) != 1 {
		t.Fatalf("expected 1 spite, got %d", len(resp.Spites))
	}
	if resp.Task == nil || resp.Task.TaskId != taskSeq {
		t.Fatalf("expected task %d, got %v", taskSeq, resp.Task)
	}
}

// ---------------------------------------------------------------------------
// DB Fallback: WaitTaskContent
// ---------------------------------------------------------------------------

func TestWaitTaskContent_DBFallback(t *testing.T) {
	env := newRPCTestEnv(t)
	sessID, taskSeq := seedDBOnlyTask(t, env, "wtc-dbfb")

	resp, err := (&Server{}).WaitTaskContent(context.Background(), &clientpb.Task{
		SessionId: sessID,
		TaskId:    taskSeq,
		Need:      -1,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent DB fallback error: %v", err)
	}
	if resp == nil || resp.Spite == nil {
		t.Fatal("expected non-nil response with spite")
	}
}

// ---------------------------------------------------------------------------
// DB Fallback: WaitTaskFinish
// ---------------------------------------------------------------------------

func TestWaitTaskFinish_DBFallback(t *testing.T) {
	env := newRPCTestEnv(t)
	sessID, taskSeq := seedDBOnlyTask(t, env, "wtf-dbfb")

	resp, err := (&Server{}).WaitTaskFinish(context.Background(), &clientpb.Task{
		SessionId: sessID,
		TaskId:    taskSeq,
	})
	if err != nil {
		t.Fatalf("WaitTaskFinish DB fallback error: %v", err)
	}
	if resp == nil || resp.Spite == nil {
		t.Fatal("expected non-nil response with spite")
	}
}

// ---------------------------------------------------------------------------
// DB Fallback: no task on disk returns error
// ---------------------------------------------------------------------------

func TestGetTaskContent_DBFallback_NoContent(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gtc-dbfb-nocontent", "gtc-dbfb-nocontent-pipe", true)

	taskPb := &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    1,
		Type:      "test-no-content",
	}
	if err := db.AddTask(taskPb); err != nil {
		t.Fatalf("AddTask: %v", err)
	}

	core.Sessions.Remove(sess.ID)

	_, err := (&Server{}).GetTaskContent(context.Background(), &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    1,
		Need:      -1,
	})
	if !errors.Is(err, types.ErrNotFoundTaskContent) {
		t.Fatalf("expected ErrNotFoundTaskContent, got %v", err)
	}
}
