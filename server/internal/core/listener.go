package core

import (
	"context"
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
	Name           string
	IP             string
	active         atomic.Bool
	readyCh        chan struct{}
	stoppedCh      chan struct{}
	readyOnce      sync.Once
	stopOnce       sync.Once
	pipelines      map[string]*clientpb.Pipeline
	pipelineMu     sync.RWMutex
	Ctrl           chan *clientpb.JobCtrl
	CtrlJob        *sync.Map
	deferredEvents sync.Map
}

// DefaultCtrlTimeout is the maximum time to wait for a listener control response.
// Kept short (5s) to prevent RPC handler starvation when a listener is disconnected.
const DefaultCtrlTimeout = 10 * time.Second

func NewListener(name, ip string) *Listener {
	l := newListener(name, ip)
	l.MarkReady()
	return l
}

func NewPendingListener(name, ip string) *Listener {
	return newListener(name, ip)
}

func newListener(name, ip string) *Listener {
	l := &Listener{
		Name:      name,
		IP:        ip,
		readyCh:   make(chan struct{}),
		stoppedCh: make(chan struct{}),
		pipelines: make(map[string]*clientpb.Pipeline),
		Ctrl:      make(chan *clientpb.JobCtrl, 8),
		CtrlJob:   &sync.Map{},
	}
	l.active.Store(true)
	return l
}

func (l *Listener) MarkReady() {
	if l == nil || l.readyCh == nil {
		return
	}
	l.readyOnce.Do(func() {
		close(l.readyCh)
	})
}

func (l *Listener) WaitReady(ctx context.Context) error {
	if l == nil {
		return errors.New("listener is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Directly constructed legacy values do not participate in the startup
	// snapshot barrier.
	if l.readyCh == nil {
		return nil
	}
	if !l.Active() {
		return errors.New("listener is stopped")
	}
	select {
	case <-l.readyCh:
		if !l.Active() {
			return errors.New("listener is stopped")
		}
		return nil
	case <-l.stoppedCh:
		return errors.New("listener is stopped")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Active returns whether the listener is active.
func (l *Listener) Active() bool {
	return l.active.Load()
}

// PushCtrl sends a control message to the listener. Returns the assigned ctrl ID.
// If the listener's Ctrl channel is full (listener not consuming), it logs a warning
// and returns 0 instead of blocking forever.
func (l *Listener) PushCtrl(ctrl *clientpb.JobCtrl) uint32 {
	return l.PushCtrlContext(context.Background(), ctrl)
}

// PushCtrlContext sends a control message and stops waiting when the caller is canceled.
func (l *Listener) PushCtrlContext(ctx context.Context, ctrl *clientpb.JobCtrl) uint32 {
	return l.pushCtrlContext(ctx, ctrl, false)
}

// PushCtrlDeferredEvent queues a control whose success event will be published
// by the caller after its authoritative state has been persisted.
func (l *Listener) PushCtrlDeferredEvent(ctx context.Context, ctrl *clientpb.JobCtrl) uint32 {
	return l.pushCtrlContext(ctx, ctrl, true)
}

func (l *Listener) pushCtrlContext(ctx context.Context, ctrl *clientpb.JobCtrl, deferEvent bool) uint32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		logs.Log.Warnf("listener %s: PushCtrl canceled before queueing: %v", l.Name, err)
		return 0
	}
	if err := l.WaitReady(ctx); err != nil {
		logs.Log.Warnf("listener %s: PushCtrl canceled before snapshot readiness: %v", l.Name, err)
		return 0
	}
	ctrl.Id = NextCtrlID()
	if deferEvent {
		l.deferredEvents.Store(ctrl.Id, struct{}{})
	}
	discardDeferredEvent := func() {
		if deferEvent {
			l.deferredEvents.Delete(ctrl.Id)
		}
	}
	timer := time.NewTimer(DefaultCtrlTimeout)
	defer timer.Stop()
	select {
	case l.Ctrl <- ctrl:
		return ctrl.Id
	case <-ctx.Done():
		discardDeferredEvent()
		logs.Log.Warnf("listener %s: PushCtrl canceled before queueing: %v", l.Name, ctx.Err())
		return 0
	case <-timer.C:
		discardDeferredEvent()
		logs.Log.Warnf("listener %s: PushCtrl timed out (channel full, listener may be disconnected)", l.Name)
		return 0
	}
}

// ConsumeDeferredEvent reports whether the control's success event is owned by
// a persistence-aware caller. The marker is retained across WaitCtrl timeouts
// until a late status arrives or the listener disconnects.
func (l *Listener) ConsumeDeferredEvent(ctrlID uint32) bool {
	_, ok := l.deferredEvents.LoadAndDelete(ctrlID)
	return ok
}

// DiscardDeferredEvent removes a marker for a control that could not be sent.
func (l *Listener) DiscardDeferredEvent(ctrlID uint32) {
	l.deferredEvents.Delete(ctrlID)
}

// WaitCtrl waits for a control response from the listener. Returns nil if the
// response does not arrive within DefaultCtrlTimeout or if ctrlID is 0 (PushCtrl failed).
func (l *Listener) WaitCtrl(i uint32) *clientpb.JobStatus {
	return l.WaitCtrlWithTimeout(i, DefaultCtrlTimeout)
}

func (l *Listener) WaitCtrlWithTimeout(i uint32, timeout time.Duration) *clientpb.JobStatus {
	if i == 0 {
		return nil
	}
	if timeout <= 0 {
		timeout = DefaultCtrlTimeout
	}
	defer l.CtrlJob.Delete(i)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		done, ok := l.CtrlJob.Load(i)
		if ok && done != nil {
			return done.(*clientpb.JobStatus)
		}
		time.Sleep(100 * time.Millisecond)
	}
	logs.Log.Warnf("listener %s: WaitCtrl(%d) timed out after %v", l.Name, i, timeout)
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

// Remove removes the exact listener instance and publishes its stopped state.
// A stale stream must never delete a replacement registered under the same name.
func (l *listeners) Remove(listener *Listener) bool {
	if listener == nil || !l.CompareAndDelete(listener.Name, listener) {
		return false
	}
	listener.stop()
	EventBroker.Publish(Event{
		EventType: consts.EventListener,
		Op:        consts.CtrlListenerStop,
		Listener:  listener.ToProtobuf(),
		Important: true,
		Message:   fmt.Sprintf("listener %s stopped", listener.Name),
	})
	return true
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

// FindByRemAgent searches all listeners for a REM pipeline that contains the given agent ID.
func (l *listeners) FindByRemAgent(agentID string) (*clientpb.Pipeline, bool) {
	var pipe *clientpb.Pipeline
	l.Range(func(key, value interface{}) bool {
		for _, p := range value.(*Listener).AllPipelines() {
			if rem := p.GetRem(); rem != nil {
				if _, ok := rem.Agents[agentID]; ok {
					pipe = p
					return false
				}
			}
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
	val.(*Listener).stop()
	return nil
}

func (l *Listener) stop() {
	l.active.Store(false)
	if l.stoppedCh != nil {
		l.stopOnce.Do(func() {
			close(l.stoppedCh)
		})
	}
	l.deferredEvents.Range(func(key, _ any) bool {
		l.deferredEvents.Delete(key)
		return true
	})

	// Clean up all pipelines and their associated jobs.
	for _, pipe := range l.AllPipelines() {
		Jobs.Remove(pipe.ListenerId, pipe.Name)
	}
	l.pipelineMu.Lock()
	l.pipelines = make(map[string]*clientpb.Pipeline)
	l.pipelineMu.Unlock()
}
