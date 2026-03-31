package core

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
)

var (
	Listeners = listeners{
		&sync.Map{},
	}
)

type Listener struct {
	Name       string
	IP         string
	active     atomic.Bool
	pipelines  map[string]*clientpb.Pipeline
	pipelineMu sync.RWMutex
	Ctrl       chan *clientpb.JobCtrl
	CtrlJob    *sync.Map
}

// DefaultCtrlTimeout is the maximum time to wait for a listener control response.
// Kept short (5s) to prevent RPC handler starvation when a listener is disconnected.
const DefaultCtrlTimeout = 10 * time.Second

func NewListener(name, ip string) *Listener {
	l := &Listener{
		Name:      name,
		IP:        ip,
		pipelines: make(map[string]*clientpb.Pipeline),
		Ctrl:      make(chan *clientpb.JobCtrl, 8),
		CtrlJob:   &sync.Map{},
	}
	l.active.Store(true)
	return l
}

// Active returns whether the listener is active.
func (l *Listener) Active() bool {
	return l.active.Load()
}

// PushCtrl sends a control message to the listener. Returns the assigned ctrl ID.
// If the listener's Ctrl channel is full (listener not consuming), it logs a warning
// and returns 0 instead of blocking forever.
func (l *Listener) PushCtrl(ctrl *clientpb.JobCtrl) uint32 {
	ctrl.Id = NextCtrlID()
	select {
	case l.Ctrl <- ctrl:
		return ctrl.Id
	case <-time.After(DefaultCtrlTimeout):
		logs.Log.Warnf("listener %s: PushCtrl timed out (channel full, listener may be disconnected)", l.Name)
		return 0
	}
}

// WaitCtrl waits for a control response from the listener. Returns nil if the
// response does not arrive within DefaultCtrlTimeout or if ctrlID is 0 (PushCtrl failed).
func (l *Listener) WaitCtrl(i uint32) *clientpb.JobStatus {
	if i == 0 {
		return nil
	}
	defer l.CtrlJob.Delete(i)
	deadline := time.Now().Add(DefaultCtrlTimeout)
	for time.Now().Before(deadline) {
		done, ok := l.CtrlJob.Load(i)
		if ok && done != nil {
			return done.(*clientpb.JobStatus)
		}
		time.Sleep(100 * time.Millisecond)
	}
	logs.Log.Warnf("listener %s: WaitCtrl(%d) timed out after %v", l.Name, i, DefaultCtrlTimeout)
	return nil
}

func (l *Listener) AddPipeline(pipeline *clientpb.Pipeline) {
	pipeline.Ip = l.IP
	l.pipelineMu.Lock()
	l.pipelines[pipeline.Name] = pipeline
	l.pipelineMu.Unlock()
}

func (l *Listener) RemovePipeline(pipeline *clientpb.Pipeline) {
	Jobs.Remove(pipeline.ListenerId, pipeline.Name)
	l.pipelineMu.Lock()
	delete(l.pipelines, pipeline.Name)
	l.pipelineMu.Unlock()
}

func (l *Listener) GetPipeline(name string) *clientpb.Pipeline {
	l.pipelineMu.RLock()
	defer l.pipelineMu.RUnlock()
	return l.pipelines[name]
}

func (l *Listener) AllPipelines() []*clientpb.Pipeline {
	l.pipelineMu.RLock()
	defer l.pipelineMu.RUnlock()
	pipelines := make([]*clientpb.Pipeline, 0, len(l.pipelines))
	for _, pipeline := range l.pipelines {
		pipelines = append(pipelines, pipeline)
	}
	return pipelines
}

func (l *Listener) ToProtobuf() *clientpb.Listener {
	return &clientpb.Listener{
		Id:        l.Name,
		Ip:        l.IP,
		Active:    l.active.Load(),
		Pipelines: &clientpb.Pipelines{Pipelines: l.AllPipelines()},
	}
}

type listeners struct {
	*sync.Map
}

func (l *listeners) Add(listener *Listener) {
	l.Store(listener.Name, listener)
	EventBroker.Publish(Event{
		EventType: consts.EventListener,
		Op:        consts.CtrlListenerStart,
		Listener:  listener.ToProtobuf(),
		Important: true,
		Message:   fmt.Sprintf("listener %s started", listener.Name),
	})
}

// Remove - Remove a listener
func (l *listeners) Remove(listener *Listener) {
	_, ok := l.LoadAndDelete(listener.Name)
	if ok {
		EventBroker.Publish(Event{
			EventType: consts.EventListener,
			Op:        consts.CtrlListenerStop,
			Listener:  listener.ToProtobuf(),
			Important: true,
			Message:   fmt.Sprintf("listener %s stopped", listener.Name),
		})
	}
}

func (l *listeners) Find(pid string) (*clientpb.Pipeline, bool) {
	var pipe *clientpb.Pipeline
	l.Range(func(key, value interface{}) bool {
		if pipe = value.(*Listener).GetPipeline(pid); pipe != nil {
			return false
		}
		return true
	})
	if pipe != nil {
		return pipe, true
	}
	return nil, false
}

func (l *listeners) FindByListener(listenerID, pid string) (*clientpb.Pipeline, bool) {
	if listenerID == "" || pid == "" {
		return nil, false
	}
	val, ok := l.Load(listenerID)
	if !ok || val == nil {
		return nil, false
	}
	pipe := val.(*Listener).GetPipeline(pid)
	if pipe == nil {
		return nil, false
	}
	return pipe, true
}

// Get - Get a Job
func (l *listeners) Get(name string) (*Listener, error) {
	if name == "" {
		return nil, types.ErrNotFoundListener
	}
	val, ok := l.Load(name)
	if ok {
		return val.(*Listener), nil
	}
	return nil, types.ErrNotFoundListener
}

func (l *listeners) PushCtrl(ctrl string, pipeline *clientpb.Pipeline) {
	val, err := l.Get(pipeline.ListenerId)
	if err == nil {
		val.PushCtrl(&clientpb.JobCtrl{
			Ctrl: ctrl,
			Job: &clientpb.Job{
				Name:     pipeline.Name,
				Pipeline: pipeline,
			},
		})
	}
}

func (l *listeners) AddPipeline(pipeline *clientpb.Pipeline) bool {
	val, err := l.Get(pipeline.ListenerId)
	if err == nil {
		val.AddPipeline(pipeline)
		return true
	}
	return false
}

func (l *listeners) RemovePipeline(pipeline *clientpb.Pipeline) bool {
	val, err := l.Get(pipeline.ListenerId)
	if err == nil {
		val.RemovePipeline(pipeline)
		return true
	}
	return false
}

func (l *listeners) ToProtobuf() *clientpb.Listeners {
	listeners := &clientpb.Listeners{}
	l.Range(func(key, value interface{}) bool {
		listeners.Listeners = append(listeners.Listeners, value.(*Listener).ToProtobuf())
		return true
	})
	return listeners
}

// Stop deactivates a listener and cleans up its pipelines and associated jobs.
func (l *listeners) Stop(name string) error {
	val, ok := l.Load(name)
	if !ok {
		return errors.New("listener not found")
	}
	listener := val.(*Listener)
	listener.active.Store(false)

	// Clean up all pipelines and their associated jobs.
	for _, pipe := range listener.AllPipelines() {
		Jobs.Remove(pipe.ListenerId, pipe.Name)
	}
	listener.pipelineMu.Lock()
	listener.pipelines = make(map[string]*clientpb.Pipeline)
	listener.pipelineMu.Unlock()

	return nil
}
