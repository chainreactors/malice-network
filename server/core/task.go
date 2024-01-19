package core

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
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
	Spite     *commonpb.Spite
	done      chan bool
	end       chan struct{}
}

func (t *Task) Handler() {
	for ok := range t.done {
		if !ok {
			return
		}
		t.Cur++
		if t.Cur == t.Total {
			close(t.done)
		}
		EventBroker.Publish(Event{
			EventType: consts.EventTaskDone,
			Task:      t,
		})
	}
	t.Finish()
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

func (t *Task) Done() {
	t.done <- true
}

func (t *Task) Finish() {
	if t.Callback != nil {
		t.Callback()
	}
}

func (t *Task) Close() {
	close(t.done)
}
