package core

import (
	"errors"
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
	Host      string
	Active    bool
	Pipelines clientpb.Pipelines
}

func (l *Listener) ToProtobuf() *clientpb.Listener {
	return &clientpb.Listener{
		Id:        l.Name,
		Addr:      l.Host,
		Active:    l.Active,
		Pipelines: &l.Pipelines,
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

// Get - Get a Job
func (l *listeners) Get(name string) *Listener {
	if name == "" {
		return nil
	}
	val, ok := l.Load(name)
	if ok {
		return val.(*Listener)
	}
	return nil
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
