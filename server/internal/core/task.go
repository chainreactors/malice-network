package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// test-swappable DB functions (overridden in task_runtime_test.go)
var (
	taskDBGetBySessionAndSeq = func(sessionID string, seq uint32) (*models.Task, error) {
		return db.GetTaskBySessionAndSeq(sessionID, seq)
	}
	taskDBUpdate = func(task *clientpb.Task) error {
		return db.UpdateTask(task)
	}
	taskDBUpdateCur = func(taskID string, cur int) error {
		return db.UpdateTaskCur(taskID, cur)
	}
	taskDBUpdateFinish = func(taskID string) error {
		return db.UpdateTaskFinish(taskID)
	}
)

func NewTasks() *Tasks {
	return &Tasks{active: &sync.Map{}}
}

type Tasks struct {
	active *sync.Map
}

// All - Return a list of all tasks
func (t *Tasks) All() []*Task {
	all := []*Task{}
	t.active.Range(func(key, value interface{}) bool {
		all = append(all, value.(*Task))
		return true
	})
	return all
}

func (t *Tasks) ToProtobuf() *clientpb.Tasks {
	tasks := &clientpb.Tasks{
		Tasks: []*clientpb.Task{},
	}
	for _, task := range t.All() {
		tasks.Tasks = append(tasks.Tasks, task.ToProtobuf())
	}
	return tasks
}

// get
func (t *Tasks) Get(taskID uint32) *Task {
	val, ok := t.active.Load(taskID)
	if ok {
		return val.(*Task)
	}
	return nil
}

// GetOrRecover 先从内存查找，找不到则回退到 DB 恢复
func (t *Tasks) GetOrRecover(sess *Session, taskID uint32) *Task {
	if task := t.Get(taskID); task != nil {
		return task
	}
	dbTask, err := taskDBGetBySessionAndSeq(sess.ID, taskID)
	if err != nil {
		return nil
	}
	recovered := FromTaskProtobuf(dbTask.ToProtobuf())
	recovered.Session = sess
	parentCtx := context.Background()
	if sess != nil && sess.Ctx != nil {
		parentCtx = sess.Ctx
	}
	recovered.Ctx, recovered.Cancel = context.WithCancel(parentCtx)
	recovered.DoneCh = make(chan bool, 1)
	if recovered.Finished() {
		recovered.closed.Store(true)
		close(recovered.DoneCh)
		recovered.Cancel()
	}
	t.Add(recovered)
	return recovered
}

func (t *Tasks) Add(task *Task) {
	t.active.Store(task.Id, task)
}

func (t *Tasks) Remove(taskId uint32) {
	t.active.Delete(taskId)
}

func (t *Tasks) GetNotFinish() []*clientpb.Task {
	var all []*clientpb.Task
	t.active.Range(func(key, value interface{}) bool {
		task := value.(*Task)
		if !task.Finished() {
			all = append(all, task.ToProtobuf())
		}
		return true
	})
	return all
}

type Task struct {
	Id         uint32
	Type       string
	SessionId  string
	Callee     string
	Cur        int
	Total      int
	Callback   func()
	Ctx        context.Context
	Cancel     context.CancelFunc
	Session    *Session
	DoneCh     chan bool
	closed     atomic.Bool
	Deadline   time.Time
	CallBy     string
	CreatedAt  time.Time
	FinishedAt time.Time
	progressMu sync.RWMutex
	closeOnce  sync.Once
}

func (t *Task) IsClosed() bool {
	return t.closed.Load()
}

func (t *Task) TaskID() string {
	return fmt.Sprintf("%s-%d", t.SessionId, t.Id)
}

func (t *Task) ToProtobuf() *clientpb.Task {
	cur, total, createdAt, finishedAt := t.snapshot()
	task := &clientpb.Task{
		TaskId:     t.Id,
		SessionId:  t.SessionId,
		Type:       t.Type,
		Cur:        int32(cur),
		Total:      int32(total),
		Timeout:    t.Timeout(),
		Finished:   t.Finished(),
		Callby:     t.CallBy,
		CreatedAt:  createdAt.Unix(),
		FinishedAt: finishedAt.Unix(),
	}
	return task
}

func FromTaskProtobuf(task *clientpb.Task) *Task {
	t := &Task{
		Id:        task.TaskId,
		Type:      task.Type,
		SessionId: task.SessionId,
		Cur:       int(task.Cur),
		Total:     int(task.Total),
		CallBy:    task.Callby,
		CreatedAt: time.Unix(task.CreatedAt, 0),
	}
	// Only set FinishedAt when the protobuf carries a positive timestamp;
	// zero-time.Unix() is negative and would produce a non-zero time.Time,
	// causing Finished() to return true for tasks that never completed.
	if task.FinishedAt > 0 {
		t.FinishedAt = time.Unix(task.FinishedAt, 0)
	}
	return t
}

func (t *Task) Name() string {
	return fmt.Sprintf("%s_%v_%s", t.SessionId, t.Id, t.Type)
}
func (t *Task) String() string {
	cur, total := t.Progress()
	return fmt.Sprintf("%d/%d", cur, total)
}

func (t *Task) snapshot() (cur, total int, createdAt, finishedAt time.Time) {
	t.progressMu.RLock()
	defer t.progressMu.RUnlock()
	return t.Cur, t.Total, t.CreatedAt, t.FinishedAt
}

func (t *Task) Progress() (cur, total int) {
	t.progressMu.RLock()
	defer t.progressMu.RUnlock()
	return t.Cur, t.Total
}

func (t *Task) FinishedAtTime() time.Time {
	t.progressMu.RLock()
	defer t.progressMu.RUnlock()
	return t.FinishedAt
}

func (t *Task) Publish(op string, spite *implantpb.Spite, msg string) {
	EventBroker.Publish(Event{
		EventType: consts.EventTask,
		Op:        op,
		Task:      t.ToProtobuf(),
		Session:   t.Session.ToProtobufLite(),
		Spite:     spite,
		Message:   msg,
		Callee:    t.Callee,
	})
}
func (t *Task) Done(spite *implantpb.Spite, msg string) {
	t.progressMu.Lock()
	t.Cur++
	cur := t.Cur
	t.progressMu.Unlock()

	if err := taskDBUpdateCur(t.TaskID(), cur); err != nil {
		logs.Log.Warnf("task %s: update cur failed: %v", t.TaskID(), err)
	}
	t.Publish(consts.CtrlTaskCallback, spite, msg)
	select {
	case t.DoneCh <- true:
	default:
	}
}

func (t *Task) Finish(spite *implantpb.Spite, msg string) {
	needsUpdate := false
	t.progressMu.Lock()
	if t.Total < 0 {
		t.Total = t.Cur
		needsUpdate = true
	}
	t.FinishedAt = time.Now()
	t.progressMu.Unlock()

	if needsUpdate {
		if err := taskDBUpdate(t.ToProtobuf()); err != nil {
			logs.Log.Warnf("task %s: update failed: %v", t.TaskID(), err)
		}
	}
	t.Publish(consts.CtrlTaskFinish, spite, msg)
	if t.Callback != nil {
		t.Callback()
	}
	if err := taskDBUpdateFinish(t.TaskID()); err != nil {
		logs.Log.Warnf("task %s: update finish failed: %v", t.TaskID(), err)
	}
	select {
	case t.DoneCh <- true:
	default:
	}
}

func (t *Task) CancelTask(spite *implantpb.Spite, msg string) {
	needsUpdate := false

	t.progressMu.Lock()
	alreadyFinished := !t.FinishedAt.IsZero() || (t.Total >= 0 && t.Cur == t.Total)
	if !alreadyFinished {
		if t.Total < 0 {
			t.Total = t.Cur
		} else {
			t.Cur = t.Total
		}
		t.FinishedAt = time.Now()
		needsUpdate = true
	}
	t.progressMu.Unlock()

	if alreadyFinished {
		return
	}

	if needsUpdate {
		if err := taskDBUpdate(t.ToProtobuf()); err != nil {
			logs.Log.Warnf("task %s: update failed: %v", t.TaskID(), err)
		}
	}
	if err := taskDBUpdateFinish(t.TaskID()); err != nil {
		logs.Log.Warnf("task %s: update finish failed: %v", t.TaskID(), err)
	}

	t.Publish(consts.CtrlTaskCancel, spite, msg)
	select {
	case t.DoneCh <- true:
	default:
	}
	t.Close()
}

func (t *Task) Finished() bool {
	t.progressMu.RLock()
	defer t.progressMu.RUnlock()
	if !t.FinishedAt.IsZero() {
		return true
	}
	return t.Total >= 0 && t.Cur == t.Total
}

func (t *Task) Timeout() bool {
	return time.Now().After(t.Deadline)
}

func (t *Task) Panic(event Event) {
	EventBroker.Publish(event)
}

func (t *Task) Close() {
	t.closeOnce.Do(func() {
		t.Cancel()
		close(t.DoneCh)
		t.closed.Store(true)
		if t.Session != nil {
			t.Session.RemoveResp(t.Id)
		}
	})
}
