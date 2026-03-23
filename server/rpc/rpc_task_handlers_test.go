package rpc

import (
	"context"
	"errors"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
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
	_, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: "nonexistent",
		All:       false,
	})
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("GetTasks(missing session) error = %v, want %v", err, types.ErrNotFoundSession)
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

	// Request active-only tasks should now fail.
	_, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: sess.ID,
		All:       false,
	})
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("GetTasks after removal error = %v, want %v", err, types.ErrNotFoundSession)
	}

	// All=true still works (DB path).
	resp, err := (&Server{}).GetTasks(context.Background(), &clientpb.TaskRequest{
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
