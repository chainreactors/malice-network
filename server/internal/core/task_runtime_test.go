package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func installTaskDBMocks() func() {
	origGet := taskDBGetBySessionAndSeq
	origUpdate := taskDBUpdate
	origUpdateCur := taskDBUpdateCur
	origUpdateFinish := taskDBUpdateFinish

	taskDBGetBySessionAndSeq = func(string, uint32) (*models.Task, error) { return nil, nil }
	taskDBUpdate = func(*clientpb.Task) error { return nil }
	taskDBUpdateCur = func(string, int) error { return nil }
	taskDBUpdateFinish = func(string) error { return nil }

	return func() {
		taskDBGetBySessionAndSeq = origGet
		taskDBUpdate = origUpdate
		taskDBUpdateCur = origUpdateCur
		taskDBUpdateFinish = origUpdateFinish
	}
}

func TestTasksGetOrRecoverFinishedTaskClosesDoneChAndBindsSession(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	taskDBGetBySessionAndSeq = func(sessionID string, seq uint32) (*models.Task, error) {
		return &models.Task{
			ID:         sessionID + "-17",
			Seq:        seq,
			Type:       "finished-from-db",
			SessionID:  sessionID,
			Cur:        2,
			Total:      2,
			FinishTime: time.Now(),
		}, nil
	}

	sess := newTestSession("task-recover-finished")
	task := sess.Tasks.GetOrRecover(sess, 17)
	if task == nil {
		t.Fatal("GetOrRecover returned nil")
	}
	if task.Session != sess {
		t.Fatal("recovered task should keep a back-reference to the session")
	}
	if !task.IsClosed() {
		t.Fatal("finished recovered task should be marked closed")
	}
	if task.Ctx.Err() == nil {
		t.Fatal("finished recovered task context should be cancelled")
	}
	select {
	case _, ok := <-task.DoneCh:
		if ok {
			t.Fatal("finished recovered task DoneCh should be closed")
		}
	default:
		t.Fatal("finished recovered task DoneCh should be immediately closed")
	}
}

func TestTaskFinishWithOpenEndedTotalPersistsResolvedTotal(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	sess := newTestSession("task-open-ended")
	task := &Task{
		Id:        9,
		Type:      "stream",
		SessionId: sess.ID,
		Session:   sess,
		Cur:       3,
		Total:     -1,
		DoneCh:    make(chan bool, 1),
	}
	task.Ctx, task.Cancel = context.WithCancel(context.Background())
	defer task.Cancel()

	var (
		updatedTask  *clientpb.Task
		finishedTask string
		callbacks    int
	)
	task.Callback = func() { callbacks++ }
	taskDBUpdate = func(pb *clientpb.Task) error {
		updatedTask = pb
		return nil
	}
	taskDBUpdateFinish = func(taskID string) error {
		finishedTask = taskID
		return nil
	}

	task.Finish(&implantpb.Spite{TaskId: task.Id}, "done")

	if task.Total != task.Cur {
		t.Fatalf("task total = %d, want %d", task.Total, task.Cur)
	}
	if !task.Finished() {
		t.Fatal("task should be marked finished after Finish")
	}
	if task.FinishedAt.IsZero() {
		t.Fatal("Finish should stamp FinishedAt")
	}
	if updatedTask == nil {
		t.Fatal("Finish should persist the resolved total for open-ended tasks")
	}
	if updatedTask.Total != int32(task.Cur) {
		t.Fatalf("persisted total = %d, want %d", updatedTask.Total, task.Cur)
	}
	if finishedTask != task.TaskID() {
		t.Fatalf("finished task id = %q, want %q", finishedTask, task.TaskID())
	}
	if callbacks != 1 {
		t.Fatalf("callback count = %d, want 1", callbacks)
	}
}

func TestNewTaskDoneChannelBuffersCompletionSignal(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	sess := newTestSession("task-buffered-signal")
	sess.Ctx, sess.Cancel = context.WithCancel(context.Background())
	defer sess.Cancel()

	task := sess.NewTask("buffered", 1)
	if cap(task.DoneCh) != 1 {
		t.Fatalf("DoneCh capacity = %d, want 1", cap(task.DoneCh))
	}

	task.Done(&implantpb.Spite{TaskId: task.Id}, "ready")

	select {
	case <-task.DoneCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("completion notification should remain buffered for late waiters")
	}
}

func TestTaskCancelMarksTaskFinishedAndClosesRuntimeState(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	sess := newTestSession("task-cancel")
	task := &Task{
		Id:        12,
		Type:      "exec",
		SessionId: sess.ID,
		Session:   sess,
		Total:     1,
		DoneCh:    make(chan bool, 1),
	}
	task.Ctx, task.Cancel = context.WithCancel(context.Background())

	var (
		updatedTask  *clientpb.Task
		finishedTask string
	)
	taskDBUpdate = func(pb *clientpb.Task) error {
		updatedTask = pb
		return nil
	}
	taskDBUpdateFinish = func(taskID string) error {
		finishedTask = taskID
		return nil
	}

	task.CancelTask(&implantpb.Spite{TaskId: task.Id}, "canceled")

	if !task.Finished() {
		t.Fatal("task should be marked finished after cancel")
	}
	if task.FinishedAt.IsZero() {
		t.Fatal("cancel should stamp FinishedAt")
	}
	if !task.IsClosed() {
		t.Fatal("cancel should close runtime task")
	}
	if task.Ctx.Err() == nil {
		t.Fatal("cancel should cancel task context")
	}
	if updatedTask == nil {
		t.Fatal("cancel should persist reconciled task progress")
	}
	if updatedTask.Cur != 1 || updatedTask.Total != 1 {
		t.Fatalf("persisted progress = %d/%d, want 1/1", updatedTask.Cur, updatedTask.Total)
	}
	if finishedTask != task.TaskID() {
		t.Fatalf("finished task id = %q, want %q", finishedTask, task.TaskID())
	}
}

func TestTask_ClosedFieldRaceSafe(t *testing.T) {
	task := &Task{
		Id:        999,
		SessionId: "race-test",
		DoneCh:    make(chan bool, 1),
	}
	task.Ctx, task.Cancel = context.WithCancel(context.Background())

	var wg sync.WaitGroup
	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = task.IsClosed()
			}
		}()
	}
	// Concurrent closer
	wg.Add(1)
	go func() {
		defer wg.Done()
		task.Close()
	}()
	wg.Wait()

	if !task.IsClosed() {
		t.Fatal("task should be closed after Close()")
	}
}
