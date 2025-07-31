package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"sync"
	"time"
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
	Id        uint32
	Type      string
	SessionId string
	Callee    string
	Cur       int
	Total     int
	Callback  func()
	Ctx       context.Context
	Cancel    context.CancelFunc
	Session   *Session
	DoneCh    chan bool
	Closed    bool
	Deadline  time.Time
	CallBy    string
}

func (t *Task) TaskID() string {
	return fmt.Sprintf("%s-%d", t.SessionId, t.Id)
}

func (t *Task) ToProtobuf() *clientpb.Task {
	task := &clientpb.Task{
		TaskId:    t.Id,
		SessionId: t.SessionId,
		Type:      t.Type,
		Cur:       int32(t.Cur),
		Total:     int32(t.Total),
		Timeout:   t.Timeout(),
		Finished:  t.Finished(),
		Callby:    t.CallBy,
	}
	return task
}

func FromTaskProtobuf(task *clientpb.Task) *Task {
	return &Task{
		Id:        task.TaskId,
		Type:      task.Type,
		SessionId: task.SessionId,
		Cur:       int(task.Cur),
		Total:     int(task.Total),
		CallBy:    task.Callby,
	}
}

func (t *Task) Name() string {
	return fmt.Sprintf("%s_%v_%s", t.SessionId, t.Id, t.Type)
}
func (t *Task) String() string {
	return fmt.Sprintf("%d/%d", t.Cur, t.Total)
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
	t.Cur++
	db.UpdateTaskCur(t.TaskID(), t.Cur)
	t.Publish(consts.CtrlTaskCallback, spite, msg)
}

func (t *Task) Finish(spite *implantpb.Spite, msg string) {
	t.Publish(consts.CtrlTaskFinish, spite, msg)
	if t.Callback != nil {
		t.Callback()
	}
	db.UpdateTaskFinish(t.TaskID())
	t.Close()
}

func (t *Task) Finished() bool {
	return t.Cur == t.Total
}

func (t *Task) Timeout() bool {
	return time.Now().After(t.Deadline)
}

func (t *Task) Panic(event Event) {
	EventBroker.Publish(event)
	t.Close()
}

func (t *Task) Close() {
	t.Cancel()
	close(t.DoneCh)
	t.Closed = true
}
