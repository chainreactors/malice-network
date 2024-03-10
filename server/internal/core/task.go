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
	*SpiteCache
	done chan bool
	end  chan struct{}
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
	t.Cancel()
	if t.Callback != nil {
		t.Callback()
	}
}

func (t *Task) Close() {
	close(t.done)
}

func NewSpiteCache(size int) *SpiteCache {
	return &SpiteCache{
		messages: make(map[uint32]*implantpb.Spite),
		size:     size,
	}
}

type SpiteCache struct {
	messages map[uint32]*implantpb.Spite
	minID    uint32
	maxID    uint32
	size     int
	lock     sync.Mutex
}

func (c *SpiteCache) AddMessage(message *implantpb.Spite) uint32 {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 自动生成ID
	c.maxID++
	id := c.maxID

	if len(c.messages) == 0 {
		c.minID = id
	}

	c.messages[id] = message

	// 删除多余的消息
	c.trim()

	return id
}

func (c *SpiteCache) GetMessage(id uint32) (*implantpb.Spite, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	msg, found := c.messages[id]
	return msg, found
}

func (c *SpiteCache) GetMessages() ([]*implantpb.Spite, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	messages := make([]*implantpb.Spite, 0, len(c.messages))
	for _, msg := range c.messages {
		messages = append(messages, msg)
	}
	return messages, true
}

func (c *SpiteCache) LastMessage() (*implantpb.Spite, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	msg, found := c.messages[c.maxID]
	return msg, found
}

func (c *SpiteCache) SetSize(newSize int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.size = newSize
	c.trim()
}

// trim 删除多余的消息直到缓存大小不超过限制
func (c *SpiteCache) trim() {
	for len(c.messages) > c.size {
		delete(c.messages, c.minID)
		c.minID++ // 增加minID直到找到下一个存在的消息
		for _, exists := c.messages[c.minID]; !exists; _, exists = c.messages[c.minID] {
			c.minID++
		}
	}
}
