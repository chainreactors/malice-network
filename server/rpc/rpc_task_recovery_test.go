package rpc

import (
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func TestRecoverSessionInitializesRecoveredTasksForRuntimeUse(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-recover-runtime", "rpc-recover-runtime-pipe", false)

	task := &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    17,
		Type:      "recover-runtime",
		Cur:       1,
		Total:     3,
	}
	if err := db.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	model, err := db.FindSession(sess.ID)
	if err != nil {
		t.Fatalf("FindSession failed: %v", err)
	}
	recovered, err := core.RecoverSession(model)
	if err != nil {
		t.Fatalf("RecoverSession failed: %v", err)
	}
	t.Cleanup(recovered.Cancel)

	runtimeTask := recovered.Tasks.Get(task.TaskId)
	if runtimeTask == nil {
		t.Fatalf("recovered task %d not found", task.TaskId)
	}
	if runtimeTask.Session != recovered {
		t.Fatal("recovered task should keep a back-reference to the recovered session")
	}
	if runtimeTask.DoneCh == nil {
		t.Fatal("recovered task should have a wait channel")
	}

	recovered.Cancel()
	select {
	case <-runtimeTask.Ctx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("recovered task context should cancel with the recovered session")
	}
}

func TestGetOrRecoverBindsRecoveredTaskToSessionContext(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-get-or-recover", "rpc-get-or-recover-pipe", true)

	task := &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    23,
		Type:      "db-recovered-task",
		Cur:       0,
		Total:     2,
	}
	if err := db.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	recovered := sess.Tasks.GetOrRecover(sess, task.TaskId)
	if recovered == nil {
		t.Fatalf("GetOrRecover(%d) returned nil", task.TaskId)
	}
	if recovered.Session != sess {
		t.Fatal("GetOrRecover should bind the recovered task to the active session")
	}
	if recovered.DoneCh == nil {
		t.Fatal("GetOrRecover should initialize DoneCh for the recovered task")
	}

	sess.Cancel()
	select {
	case <-recovered.Ctx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("GetOrRecover should derive task context from session context")
	}
}
