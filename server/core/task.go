package core

import (
	"fmt"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"sync"
)

type Tasks struct {
	active     *sync.Map
	taskNumber uint32
}

// get
func (t *Tasks) Get(taskID uint32) *Task {
	val, ok := t.active.Load(taskID)
	if ok {
		return val.(*Task)
	}
	return nil
}

func (t *Tasks) nextTaskId() uint32 {
	t.taskNumber++
	return t.taskNumber
}

func (t *Tasks) Add(task *Task) {
	id := t.nextTaskId()
	task.Id = id
	t.active.Store(id, task)
}

func (t *Tasks) Remove(task *Task) {
	t.active.Delete(task.Id)
}

func NewTask(name string, total int) *Task {
	return &Task{
		Type:  name,
		Total: total,
		done:  make(chan bool),
	}
}

type Task struct {
	Id        uint32
	Type      string
	SessionId string
	done      chan bool
	Cur       int
	Total     int
}

func (t *Task) ToProtobuf() *clientpb.Task {
	return &clientpb.Task{
		TaskId:    t.Id,
		SessionId: t.SessionId,
		Type:      t.Type,
		Cur:       int32(t.Cur),
		Total:     int32(t.Total),
		Status:    0,
	}
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
	go func() {
		for ok := range t.done {
			if !ok {
				return
			}
			t.Cur++
		}
	}()
}

func (t *Task) Close() {
	close(t.done)
}
