package core

import (
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"google.golang.org/protobuf/proto"
	"sync"
)

var (
	Listeners = listeners{
		&sync.Map{},
	}
)

type Pipeline interface {
	ID() string
	Start() error
	Addr() string
	Close() error
	ToProtobuf() proto.Message
}

type Pipelines map[string]Pipeline

func (ps Pipelines) Add(p Pipeline) {
	ps[p.ID()] = p
}

func (ps Pipelines) Get(id string) Pipeline {
	return ps[id]
}

func (ps Pipelines) ToProtobuf() []*lispb.Pipeline {
	var pls []*lispb.Pipeline
	for _, p := range ps {
		pls = append(pls, types.BuildPipeline(p.ToProtobuf()))
	}
	return pls
}

type Listener struct {
	Name      string
	Host      string
	Active    bool
	Pipelines []*lispb.Pipeline
}

func (l *Listener) ToProtobuf() *clientpb.Listener {
	return &clientpb.Listener{
		Id:        l.Name,
		Addr:      l.Host,
		Active:    l.Active,
		Pipelines: l.Pipelines,
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
func (l *listeners) Get(name int) *Listener {
	if name <= 0 {
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
