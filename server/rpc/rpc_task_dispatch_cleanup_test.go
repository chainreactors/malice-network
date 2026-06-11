package rpc

import (
	"errors"
	"os"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func TestGenericHandlerRollsBackTaskWhenAddTaskFails(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-task-add-fail", "rpc-task-add-fail-pipe", true)

	startSeq := sess.Taskseq
	wantErr := errors.New("add task failed")
	oldAddTask := genericAddTask
	genericAddTask = func(task *clientpb.Task) error {
		return wantErr
	}
	t.Cleanup(func() {
		genericAddTask = oldAddTask
	})

	_, err := (&Server{}).Ping(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 7})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Ping error = %v, want %v", err, wantErr)
	}

	assertTaskDispatchRolledBack(t, sess, startSeq+1)
}

func TestGenericHandlerRollsBackTaskWhenRequestCachingFails(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-task-request-cache-fail", "rpc-task-request-cache-fail-pipe", true)

	startSeq := sess.Taskseq
	wantErr := errors.New("write request failed")
	oldWriteTaskRequest := genericWriteTaskRequest
	genericWriteTaskRequest = func(task *core.Task, spite *implantpb.Spite) error {
		return wantErr
	}
	t.Cleanup(func() {
		genericWriteTaskRequest = oldWriteTaskRequest
	})

	_, err := (&Server{}).Ping(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 8})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Ping error = %v, want %v", err, wantErr)
	}

	assertTaskDispatchRolledBack(t, sess, startSeq+1)
}

func TestGenericHandlerRollsBackTaskWhenPipelineIsUnavailable(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-task-missing-pipeline", "rpc-task-missing-pipeline-pipe", true)

	startSeq := sess.Taskseq
	pipelinesCh.Delete(sess.PipelineID)

	_, err := (&Server{}).Ping(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 9})
	if !errors.Is(err, types.ErrNotFoundPipeline) {
		t.Fatalf("Ping error = %v, want %v", err, types.ErrNotFoundPipeline)
	}

	assertTaskDispatchRolledBack(t, sess, startSeq+1)
}

func TestGenericHandlerRollsBackTaskWhenRequestDispatchFails(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-task-dispatch-fail", "rpc-task-dispatch-fail-pipe", true)

	startSeq := sess.Taskseq
	wantErr := errors.New("send failed")
	pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
		sendMsg: func(interface{}) error {
			return wantErr
		},
	})
	t.Cleanup(func() {
		pipelinesCh.Delete(sess.PipelineID)
	})

	_, err := (&Server{}).Ping(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 10})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Ping error = %v, want %v", err, wantErr)
	}

	assertTaskDispatchRolledBack(t, sess, startSeq+1)
}

func TestStreamGenericHandlerRollsBackTaskWhenPipelineIsUnavailable(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-stream-missing-pipeline", "rpc-stream-missing-pipeline-pipe", true)

	startSeq := sess.Taskseq
	pipelinesCh.Delete(sess.PipelineID)

	greq, err := newGenericRequest(incomingSessionContext(sess.ID), &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("newGenericRequest failed: %v", err)
	}

	if _, _, err = (&Server{}).StreamGenericHandler(incomingSessionContext(sess.ID), greq); !errors.Is(err, types.ErrNotFoundPipeline) {
		t.Fatalf("StreamGenericHandler error = %v, want %v", err, types.ErrNotFoundPipeline)
	}

	assertTaskDispatchRolledBack(t, sess, startSeq+1)
}

func TestStreamGenericHandlerRollsBackTaskWhenRequestDispatchFails(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-stream-dispatch-fail", "rpc-stream-dispatch-fail-pipe", true)

	startSeq := sess.Taskseq
	wantErr := errors.New("stream send failed")
	pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
		sendMsg: func(interface{}) error {
			return wantErr
		},
	})
	t.Cleanup(func() {
		pipelinesCh.Delete(sess.PipelineID)
	})

	greq, err := newGenericRequest(incomingSessionContext(sess.ID), &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("newGenericRequest failed: %v", err)
	}

	if _, _, err = (&Server{}).StreamGenericHandler(incomingSessionContext(sess.ID), greq); !errors.Is(err, wantErr) {
		t.Fatalf("StreamGenericHandler error = %v, want %v", err, wantErr)
	}

	assertTaskDispatchRolledBack(t, sess, startSeq+1)
}

func assertTaskDispatchRolledBack(t *testing.T, sess *core.Session, taskID uint32) {
	t.Helper()

	if task := sess.Tasks.Get(taskID); task != nil {
		t.Fatalf("runtime task %d still present after rollback: %#v", taskID, task)
	}

	tasks, err := db.ListTasksBySession(sess.ID)
	if err != nil {
		t.Fatalf("ListTasksBySession(%q) failed: %v", sess.ID, err)
	}
	if len(tasks) != 0 {
		t.Fatalf("db tasks = %#v, want no persisted tasks after rollback", tasks)
	}

	if _, ok := sess.GetResp(taskID); ok {
		t.Fatalf("response channel for task %d should be removed after rollback", taskID)
	}

	requestPath, err := taskRequestPath(&core.Task{SessionId: sess.ID, Id: taskID})
	if err != nil {
		t.Fatalf("taskRequestPath failed: %v", err)
	}
	if _, err := os.Stat(requestPath); !os.IsNotExist(err) {
		t.Fatalf("request cache %q still exists after rollback (err=%v)", requestPath, err)
	}
}
