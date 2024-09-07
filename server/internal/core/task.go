package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
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

func (t *Tasks) Remove(task *Task) {
	t.active.Delete(task.Id)
}

type Task struct {
	Id        uint32
	Type      string
	SessionId string
	Cur       int
	Total     int
	Callback  func()
	Ctx       context.Context
	Cancel    context.CancelFunc
	//Status    *implantpb.Spite
	DoneCh chan bool
}

func (t *Task) Handler() {
	for ok := range t.DoneCh {
		if !ok {
			return
		}
		t.Cur++
		if t.Cur == t.Total {
			t.Finish()
			return
		}
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

func (t *Task) Done(event Event) {
	EventBroker.Publish(event)
	t.DoneCh <- true
}

func (t *Task) Finish() {
	EventBroker.Publish(Event{
		Task:      t,
		EventType: consts.EventTask,
		Op:        consts.CtrlTaskFinish,
	})
	if t.Callback != nil {
		t.Callback()
	}
	t.Close()
}

func (t *Task) Panic(event Event) {
	//t.Status = status
	EventBroker.Publish(event)
	t.Close()
}

func (t *Task) Close() {
	t.Cancel()
	close(t.DoneCh)
}
