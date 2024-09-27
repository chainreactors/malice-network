package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"sync"
)

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
}

func (t *Task) Handler() {
	for ok := range t.DoneCh {
		if !ok {
			return
		}
		t.Cur++
	}
}

func (t *Task) ToProtobuf() *clientpb.Task {
	task := &clientpb.Task{
		TaskId:    t.Id,
		SessionId: t.SessionId,
		Type:      t.Type,
		Cur:       int32(t.Cur),
		Total:     int32(t.Total),
		Status:    0,
	}
	return task
}

func (t *Task) Name() string {
	return fmt.Sprintf("%s_%s", t.SessionId, t.Type)
}
func (t *Task) String() string {
	return fmt.Sprintf("%d/%d", t.Cur, t.Total)
}

func (t *Task) Percent() string {
	return fmt.Sprintf("%f/100%", t.Cur/t.Total*100)
}

func (t *Task) Publish(op string, spite *implantpb.Spite, msg string) {
	EventBroker.Publish(Event{
		EventType: consts.EventTask,
		Op:        op,
		Task:      t.ToProtobuf(),
		Session:   t.Session.ToProtobuf(),
		Spite:     spite,
		Message:   msg,
		Callee:    t.Callee,
	})
}
func (t *Task) Done(spite *implantpb.Spite, msg string) {
	t.Publish(consts.CtrlTaskCallback, spite, msg)
	t.DoneCh <- true
}

func (t *Task) Finish(spite *implantpb.Spite, msg string) {
	t.Publish(consts.CtrlTaskFinish, spite, msg)
	if t.Callback != nil {
		t.Callback()
	}
	t.Close()
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
