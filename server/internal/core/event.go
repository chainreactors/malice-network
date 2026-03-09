package core

import (
	"fmt"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	inotify "github.com/chainreactors/malice-network/server/internal/notify"
	"github.com/chainreactors/tui"
)

const (
	// Size is arbitrary, just want to avoid weird cases where we'd block on channel sends
	eventBufSize = 25
)

// format produces plain-text structured messages (no ANSI colors).
// Coloring is the responsibility of each consumer (CLI, GUI, MCP, etc.).
func (event *Event) format() string {
	clientName := ""
	if event.Client != nil {
		clientName = event.Client.Name
	}

	switch event.EventType {
	case consts.EventClient:
		if event.Op == consts.CtrlClientJoin {
			return fmt.Sprintf("%s has joined the game", clientName)
		} else if event.Op == consts.CtrlClientLeft {
			return fmt.Sprintf("%s left the game", clientName)
		}
	case consts.EventBroadcast:
		msg := fmt.Sprintf("%s : %s", clientName, event.Message)
		if event.Err != "" {
			msg += "  " + event.Err
		}
		return msg
	case consts.EventNotify:
		msg := fmt.Sprintf("%s notified: %s", clientName, event.Message)
		if event.Err != "" {
			msg += " " + event.Err
		}
		return msg
	case consts.EventListener:
		msg := fmt.Sprintf("[%s] %s: %s", event.EventType, event.Op, event.Message)
		if event.Err != "" {
			msg += " " + event.Err
		}
		return msg
	case consts.EventWebsite:
		msg := fmt.Sprintf("[%s] %s: %s", event.EventType, event.Op, event.Message)
		if event.Err != "" {
			msg += " " + event.Err
		}
		return msg
	case consts.EventBuild:
		return fmt.Sprintf("[%s] %s", event.EventType, event.Message)
	case consts.EventCert:
		return fmt.Sprintf("[%s] %s", event.EventType, event.Message)
	case consts.EventPivot:
		return fmt.Sprintf("[%s] %s: %s", event.EventType, event.Op, event.Message)
	case consts.EventContext:
		return fmt.Sprintf("[%s] %s: %s", event.EventType, event.Op, event.Message)
	case consts.EventSession:
		sid := event.Session.SessionId
		switch event.Op {
		case consts.CtrlSessionRegister:
			return fmt.Sprintf("[%s] %s", consts.CtrlSessionRegister, event.Message)
		case consts.CtrlSessionDead:
			return fmt.Sprintf("[%s] %s", consts.CtrlSessionDead, event.Message)
		case consts.CtrlSessionReborn:
			return fmt.Sprintf("[%s] %s", consts.CtrlSessionReborn, event.Message)
		case consts.CtrlSessionInit:
			return fmt.Sprintf("[%s] %s", consts.CtrlSessionInit, event.Message)
		case consts.CtrlSessionTask:
			return fmt.Sprintf("[%s.%d] run task %s: %s",
				sid, event.Task.TaskId, event.Task.Type, event.Message)
		case consts.CtrlSessionError:
			return fmt.Sprintf("[%s] task: %d error: %s",
				sid, event.Task.TaskId, event.Err)
		case consts.CtrlSessionLog:
			return fmt.Sprintf("[%s] log:\n%s", sid, event.Message)
		case consts.CtrlSessionCheckin:
			return ""
		}
	case consts.EventJob:
		if event.Err != "" {
			return fmt.Sprintf("[%s] %s: %s", event.EventType, event.Op, event.Err)
		}
		pipeline := event.Job.GetPipeline()
		kvView := func(pipeType string) string {
			return fmt.Sprintf("[%s] %s: %s \n%s", event.EventType, event.Op, pipeType,
				tui.NewOrderedKVTable(pipeline.KVMap()).View())
		}
		switch pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			return kvView("tcp")
		case *clientpb.Pipeline_Bind:
			return kvView("bind")
		case *clientpb.Pipeline_Http:
			if event.Op == consts.CtrlAcme {
				return fmt.Sprintf("[%s] %s: cert %s create success", event.EventType, event.Op,
					pipeline.Tls.Domain)
			}
			return kvView("http")
		case *clientpb.Pipeline_Rem:
			if event.Op == consts.CtrlRemAgentLog {
				return ""
			}
			return kvView("rem")
		case *clientpb.Pipeline_Web:
			baseURL := pipeline.URL()
			if event.Op == consts.CtrlWebContentAddArtifact {
				if cont := event.Job.FirstContent(); cont != nil {
					return fmt.Sprintf("[%s] %s: artifact %s amount at %s", event.EventType, event.Op,
						cont.Id, baseURL+cont.Path)
				}
			} else if event.Op == consts.CtrlWebContentAdd {
				if cont := event.Job.FirstContent(); cont != nil {
					return fmt.Sprintf("[%s] %s: content add success, path: %s",
						event.EventType, event.Op, baseURL+cont.Path)
				}
			}
			return kvView("web")
		}
	}
	return event.Message
}

type Event struct {
	Session *clientpb.Session
	Job     *clientpb.Job
	Client  *clientpb.Client
	Task    *clientpb.Task
	Spite   *implantpb.Spite

	Important bool
	EventType string
	Op        string
	Message   string
	Err       string
	Callee    string
	IsNotify  bool
}

func (event *Event) String() string {
	var id string

	if event.Job != nil {
		id = fmt.Sprintf("Job %d %s", event.Job.Id, event.Job.Name)
	} else if event.Task != nil {
		id = fmt.Sprintf("Task %s %d", event.Task.SessionId, event.Task.TaskId)
	} else if event.Session != nil {
		id = fmt.Sprintf("Session %s", event.Session.SessionId)
	}
	if event.Err != "" {
		return fmt.Sprintf("%s %s: %s", id, event.Op, event.Err)
	} else {
		return fmt.Sprintf("%s %s: %s", id, event.Op, event.Message)
	}
}

// toprotobuf
func (event *Event) ToProtobuf() *clientpb.Event {
	return &clientpb.Event{
		Session:   event.Session,
		Job:       event.Job,
		Client:    event.Client,
		Task:      event.Task,
		Spite:     event.Spite,
		Type:      event.EventType,
		Op:        event.Op,
		Formatted: event.format(),
		Message:   []byte(event.Message),
		Err:       event.Err,
		Callee:    event.Callee,
	}
}

type eventBroker struct {
	stop        chan struct{}
	publish     chan Event
	subscribe   chan chan Event
	unsubscribe chan chan Event
	send        chan Event
	notifier    inotify.Notifier

	lock  *sync.Mutex
	cache *RingCache
}

func (broker *eventBroker) Start() {
	subscribers := map[chan Event]struct{}{}
	for {
		select {
		case <-broker.stop:
			for sub := range subscribers {
				close(sub)
			}
			return
		case sub := <-broker.subscribe:
			subscribers[sub] = struct{}{}
		case sub := <-broker.unsubscribe:
			delete(subscribers, sub)
		case event := <-broker.publish:
			if event.Important {
				logs.Log.Infof("[event.%s] %s", event.EventType, event.String())
			} else if event.EventType != consts.EventHeartbeat {
				logs.Log.Debugf("[event.%s] %s", event.EventType, event.String())
			}
			broker.lock.Lock()
			for sub := range subscribers {
				sub <- event
			}
			broker.lock.Unlock()
		}
	}
}

// Stop - Close the broker channel
func (broker *eventBroker) Stop() {
	close(broker.stop)
}

// Subscribe - Generate a new subscription channel
func (broker *eventBroker) Subscribe() chan Event {
	events := make(chan Event, eventBufSize)
	broker.subscribe <- events
	return events
}

// Unsubscribe - Remove a subscription channel
func (broker *eventBroker) Unsubscribe(events chan Event) {
	broker.unsubscribe <- events
	//close(events)
}

// Publish - Push a message to all subscribers
func (broker *eventBroker) Publish(event Event) {
	if event.Important {
		broker.cache.Add(&event)
	}
	broker.publish <- event
	if event.IsNotify {
		broker.Notify(event)
	}
}

func (broker *eventBroker) GetAll() []*Event {
	var events []*Event
	for _, v := range broker.cache.GetAll() {
		events = append(events, v.(*Event))
	}

	return events
}

// Notify - Notify all third-patry services
func (broker *eventBroker) Notify(event Event) {
	SafeGo(func() { broker.notifier.Send(event.EventType, event.Op, event.Message) })
}

func NewBroker() *eventBroker {
	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
		lock:        &sync.Mutex{},
	}
	SafeGo(broker.Start)
	ticker := GlobalTicker

	publishHeartbeat := func(interval string) {
		broker.Publish(Event{
			EventType: consts.EventHeartbeat,
			Op:        interval,
			Message:   fmt.Sprintf("Heartbeat event every %s", interval),
			IsNotify:  false,
		})
	}

	ticker.Start(1, func() { publishHeartbeat(consts.CtrlHeartbeat1s) })
	ticker.Start(5, func() { publishHeartbeat(consts.CtrlHeartbeat5s) })
	ticker.Start(10, func() { publishHeartbeat(consts.CtrlHeartbeat10s) })
	ticker.Start(15, func() { publishHeartbeat(consts.CtrlHeartbeat15s) })
	ticker.Start(30, func() { publishHeartbeat(consts.CtrlHeartbeat30s) })
	ticker.Start(60, func() { publishHeartbeat(consts.CtrlHeartbeat1m) })
	ticker.Start(300, func() { publishHeartbeat(consts.CtrlHeartbeat5m) })
	ticker.Start(600, func() { publishHeartbeat(consts.CtrlHeartbeat10m) })
	ticker.Start(900, func() { publishHeartbeat(consts.CtrlHeartbeat15m) })
	ticker.Start(1200, func() { publishHeartbeat(consts.CtrlHeartbeat20m) })
	ticker.Start(1800, func() { publishHeartbeat(consts.CtrlHeartbeat30m) })
	ticker.Start(3600, func() { publishHeartbeat(consts.CtrlHeartbeat60m) })
	EventBroker = broker
	return broker
}

var (
	// EventBroker - Distributes event messages
	EventBroker *eventBroker
)

func (broker *eventBroker) InitService(config *configs.NotifyConfig) error {
	return broker.notifier.InitService(config)
}
