package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func installTaskDBMocks() func() {
	origGet := taskDBGetBySessionAndSeq
	origUpdate := taskDBUpdate
	origUpdateCur := taskDBUpdateCur
	origUpdateTotal := taskDBUpdateTotal
	origUpdateFinish := taskDBUpdateFinish

	taskDBGetBySessionAndSeq = func(string, uint32) (*models.Task, error) { return nil, nil }
	taskDBUpdate = func(*clientpb.Task) error { return nil }
	taskDBUpdateCur = func(string, int) error { return nil }
	taskDBUpdateTotal = func(string, int) error { return nil }
	taskDBUpdateFinish = func(string) error { return nil }

	return func() {
		taskDBGetBySessionAndSeq = origGet
		taskDBUpdate = origUpdate
		taskDBUpdateCur = origUpdateCur
		taskDBUpdateTotal = origUpdateTotal
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

func TestTaskFailMarksTerminalStateAndPersistsReason(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	oldBroker := EventBroker
	EventBroker = newTestBroker()
	defer func() { EventBroker = oldBroker }()

	sess := newTestSession("task-terminal-failure")
	task := sess.NewTask("exec", 1)
	sess.StoreResp(task.Id, make(chan *implantpb.Spite, 1))

	var (
		updatedTask  *clientpb.Task
		finishedTask string
		updateCount  int
	)
	taskDBUpdate = func(pb *clientpb.Task) error {
		updatedTask = pb
		updateCount++
		return nil
	}
	taskDBUpdateFinish = func(taskID string) error {
		finishedTask = taskID
		return nil
	}

	task.Fail(&implantpb.Spite{TaskId: task.Id}, "delivery outcome is ambiguous")
	task.Fail(nil, "duplicate failure")

	if !task.Finished() || task.FinishedAtTime().IsZero() {
		t.Fatal("failed task should have a terminal timestamp")
	}
	if !task.IsClosed() || task.Ctx.Err() == nil {
		t.Fatal("failed task should close its runtime context")
	}
	if _, ok := sess.GetResp(task.Id); ok {
		t.Fatal("failed task response channel should be removed")
	}
	got := task.ToProtobuf()
	if got.Status != consts.CtrlStatusFailed || got.Error != "delivery outcome is ambiguous" {
		t.Fatalf("failed task state = status %d error %q", got.Status, got.Error)
	}
	if updatedTask == nil || updatedTask.Status != consts.CtrlStatusFailed || updatedTask.Error != got.Error {
		t.Fatalf("persisted failed task = %#v", updatedTask)
	}
	if finishedTask != task.TaskID() {
		t.Fatalf("finished task id = %q, want %q", finishedTask, task.TaskID())
	}
	if updateCount != 1 {
		t.Fatalf("terminal state persisted %d times, want 1", updateCount)
	}
}

func TestTaskDoneDoesNotSendAfterConcurrentFailureClosesChannel(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	oldBroker := EventBroker
	EventBroker = nil
	defer func() { EventBroker = oldBroker }()

	updateStarted := make(chan struct{})
	releaseUpdate := make(chan struct{})
	taskDBUpdateCur = func(string, int) error {
		close(updateStarted)
		<-releaseUpdate
		return nil
	}

	task := &Task{
		Id:        22,
		SessionId: "task-done-fail-race",
		Total:     2,
		DoneCh:    make(chan bool, 1),
	}
	task.Ctx, task.Cancel = context.WithCancel(context.Background())

	panicValue := make(chan interface{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { panicValue <- recover() }()
		task.Done(&implantpb.Spite{TaskId: task.Id}, "chunk")
	}()

	select {
	case <-updateStarted:
	case <-time.After(time.Second):
		t.Fatal("Task.Done did not reach the persistence barrier")
	}
	task.Fail(nil, "task deadline exceeded")
	close(releaseUpdate)
	<-done

	if got := <-panicValue; got != nil {
		t.Fatalf("Task.Done panicked after concurrent Task.Fail: %v", got)
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

func TestTaskExtendDeadlineIsRaceSafe(t *testing.T) {
	task := &Task{Deadline: time.Now().Add(-time.Second)}
	if !task.Timeout() {
		t.Fatal("past deadline should be timed out")
	}

	future := time.Now().Add(time.Hour)
	task.ExtendDeadline(future)
	if task.Timeout() {
		t.Fatal("extended future deadline should not be timed out")
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				task.ExtendDeadline(time.Now().Add(time.Hour))
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = task.Timeout()
			}
		}()
	}
	wg.Wait()
}

func TestUpdateTotalPersistsOnlyTotalField(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	var (
		persistedID    string
		persistedTotal int
		curWritten     bool
	)
	taskDBUpdateTotal = func(taskID string, total int) error {
		persistedID = taskID
		persistedTotal = total
		return nil
	}
	// Ensure taskDBUpdate (which writes cur+total) is NOT called.
	taskDBUpdate = func(pb *clientpb.Task) error {
		curWritten = true
		return nil
	}

	task := &Task{
		Id:        5,
		SessionId: "update-total-test",
		Cur:       3,
		Total:     1,
	}

	task.UpdateTotal(10)

	if task.Total != 10 {
		t.Fatalf("in-memory total = %d, want 10", task.Total)
	}
	if persistedID != task.TaskID() {
		t.Fatalf("persisted task id = %q, want %q", persistedID, task.TaskID())
	}
	if persistedTotal != 10 {
		t.Fatalf("persisted total = %d, want 10", persistedTotal)
	}
	if curWritten {
		t.Fatal("UpdateTotal should not call taskDBUpdate (which writes cur)")
	}
}

func TestUpdateTotalDoesNotRaceWithDone(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	sess := newTestSession("update-total-race")
	sess.Ctx, sess.Cancel = context.WithCancel(context.Background())
	defer sess.Cancel()

	task := sess.NewTask("download", 1)

	// Run UpdateTotal and Done concurrently.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		task.UpdateTotal(10)
	}()
	go func() {
		defer wg.Done()
		task.Done(&implantpb.Spite{TaskId: task.Id}, "chunk")
	}()
	wg.Wait()

	cur, total := task.Progress()
	if total != 10 {
		t.Fatalf("total = %d, want 10", total)
	}
	if cur != 1 {
		t.Fatalf("cur = %d, want 1", cur)
	}
}

func TestFinishDoesNotOverwritePositiveTotal(t *testing.T) {
	cleanup := installTaskDBMocks()
	defer cleanup()

	broker := newTestBroker()
	oldBroker := EventBroker
	EventBroker = broker
	defer func() { EventBroker = oldBroker }()

	sess := newTestSession("finish-positive-total")
	task := &Task{
		Id:        7,
		Type:      "download",
		SessionId: sess.ID,
		Session:   sess,
		Cur:       5,
		Total:     10,
		DoneCh:    make(chan bool, 1),
	}
	task.Ctx, task.Cancel = context.WithCancel(context.Background())
	defer task.Cancel()

	// taskDBUpdate should NOT be called when Total is already positive,
	// because the `if t.Total < 0` branch in Finish() won't trigger.
	var updateCalled bool
	taskDBUpdate = func(pb *clientpb.Task) error {
		updateCalled = true
		return nil
	}

	task.Finish(&implantpb.Spite{TaskId: task.Id}, "done")

	if task.Total != 10 {
		t.Fatalf("total = %d, want 10 (should not be overwritten)", task.Total)
	}
	if updateCalled {
		t.Fatal("Finish should not call taskDBUpdate when Total is already positive")
	}
}
