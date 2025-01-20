package core

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"sync"
)

var (
	Listeners = listeners{
		&sync.Map{},
	}
)

type Listener struct {
	Name      string
	IP        string
	Active    bool
	Pipelines map[string]*clientpb.Pipeline
	Ctrl      chan *clientpb.JobCtrl
}

func (l *Listener) PushCtrl(ctrl *clientpb.JobCtrl) {
	l.Ctrl <- ctrl
}

func (l *Listener) AddPipeline(pipeline *clientpb.Pipeline) {
	l.Pipelines[pipeline.Name] = pipeline
}

func (l *Listener) RemovePipeline(pipeline *clientpb.Pipeline) {
	delete(l.Pipelines, pipeline.Name)
}

func (l *Listener) GetPipeline(name string) *clientpb.Pipeline {
	return l.Pipelines[name]
}

func (l *Listener) AllPipelines() []*clientpb.Pipeline {
	pipelines := []*clientpb.Pipeline{}
	for _, pipeline := range l.Pipelines {
		pipelines = append(pipelines, pipeline)
	}
	return pipelines
}

func (l *Listener) ToProtobuf() *clientpb.Listener {
	return &clientpb.Listener{
		Id:        l.Name,
		Ip:        l.IP,
		Active:    l.Active,
		Pipelines: &clientpb.Pipelines{Pipelines: l.AllPipelines()},
	}
}

type listeners struct {
	*sync.Map
}

func (l *listeners) Add(listener *Listener) {
	l.Store(listener.Name, listener)
	//EventBroker.Publish(Event{
	//	Job:       listener,
	//	EventType: consts.JobStartedEvent,
	//})
}

// Remove - Remove a job
func (l *listeners) Remove(listener *Listener) {
	_, _ = l.LoadAndDelete(listener.Name)
	//if ok {
	//	EventBroker.Publish(Event{
	//		Job:       listener,
	//		EventType: consts.JobStoppedEvent,
	//	})
	//}
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

// Get - Get a Job
func (l *listeners) Get(name string) (*Listener, error) {
	if name == "" {
		return nil, errs.ErrNotFoundListener
	}
	val, ok := l.Load(name)
	if ok {
		return val.(*Listener), nil
	}
	return nil, errs.ErrNotFoundListener
}

func (l *listeners) PushCtrl(ctrl string, pipeline *clientpb.Pipeline) {
	val, err := l.Get(pipeline.ListenerId)
	if err == nil {
		val.PushCtrl(&clientpb.JobCtrl{
			Id:   NextCtrlID(),
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

// Stop - Stop a listener
func (l *listeners) Stop(name string) error {
	val, ok := l.Load(name)
	if ok {
		val.(*Listener).Active = false
		//for _, pipeline := range val.(*Listener).Pipelines.Pipelines {
		//	// TODO close pipeline
		//	//err := pipeline.Close()
		//	if err != nil {
		//		// TODO - need or not give error if pipeline close failed
		//		continue
		//	}
		//}
	} else {
		return errors.New("listener not found")
	}
	return nil
}
