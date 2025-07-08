package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/dingding"
	"github.com/nikoksr/notify/service/http"
	"github.com/nikoksr/notify/service/lark"
	"github.com/nikoksr/notify/service/telegram"
	"net/url"
	"path"
	"strings"
	"sync"
)

const (
	// Size is arbitrary, just want to avoid weird cases where we'd block on channel sends
	eventBufSize = 25
)

func (event *Event) format() string {
	switch event.EventType {
	case consts.EventClient:
		if event.Op == consts.CtrlClientJoin {
			return fmt.Sprintf("%s has joined the game", event.Client.Name)
		} else if event.Op == consts.CtrlClientLeft {
			return fmt.Sprintf("%s left the game", event.Client.Name)
		}
	case consts.EventBroadcast:
		return fmt.Sprintf("%s : %s  %s", event.Client.Name, event.Message, event.Err)
	case consts.EventNotify:
		return fmt.Sprintf("%s notified: %s %s", event.Client.Name, event.Message, event.Err)
	case consts.EventListener:
		return fmt.Sprintf("[%s] %s: %s %s", event.EventType, event.Op, event.Message, event.Err)
	case consts.EventWebsite:
		return fmt.Sprintf("[%s] %s: %s %s", event.EventType, event.Op, event.Message, event.Err)
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
			return logs.GreenBold(fmt.Sprintf("[%s]: %s", consts.CtrlSessionRegister, event.Message))
		case consts.CtrlSessionTask:
			return logs.GreenBold(fmt.Sprintf("[%s.%d] run task %s: %s", sid, event.Task.TaskId, event.Task.Type, event.Message))
		case consts.CtrlSessionError:
			return logs.GreenBold(fmt.Sprintf("[%s] task: %d error: %s", sid, event.Task.TaskId, event.Err))
		case consts.CtrlSessionLog:
			return fmt.Sprintf("[%s] log: \n%s", sid, event.Message)
		}
	case consts.EventJob:
		if event.Err != "" {
			return fmt.Sprintf("[%s] %s: %s", event.EventType, event.Op, event.Err)
		}
		pipeline := event.Job.GetPipeline()
		switch pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			return fmt.Sprintf("[%s] %s: tcp %s on %s %s:%d", event.EventType, event.Op,
				pipeline.Name, pipeline.ListenerId, pipeline.Ip, pipeline.GetTcp().Port)
		case *clientpb.Pipeline_Bind:
			return fmt.Sprintf("[%s] %s: bind %s on %s %s", event.EventType, event.Op,
				pipeline.Name, pipeline.ListenerId, pipeline.Ip)
		case *clientpb.Pipeline_Http:
			if event.Op == consts.CtrlAcme {
				return fmt.Sprintf("[%s] %s: cert %s create success", event.EventType, event.Op,
					pipeline.Tls.Domain)
			}
			return fmt.Sprintf("[%s] %s: http %s on %s %s:%d", event.EventType, event.Op,
				pipeline.Name, pipeline.ListenerId, pipeline.Ip, pipeline.GetHttp().Port)
		case *clientpb.Pipeline_Rem:
			if event.Op == consts.CtrlRemAgentLog {
				return ""
			}
			return fmt.Sprintf("[%s] %s: rem %s on %s %s:%d", event.EventType, event.Op,
				pipeline.Name, pipeline.ListenerId, pipeline.Ip, pipeline.GetRem().Port)
		case *clientpb.Pipeline_Web:
			scheme := "http"
			if pipeline.Tls.Enable {
				scheme = "https"
			}
			web := pipeline.GetWeb()
			// baseURL 只到 host:port
			baseURL := fmt.Sprintf("%s://%s:%d", scheme, pipeline.Ip, web.Port)

			if event.Op == consts.CtrlWebContentAddArtifact {
				routePath := path.Join(web.Root, event.Job.Contents[pipeline.ListenerId].Path)
				return fmt.Sprintf("[%s] %s: artifact amount at %s/%s", event.EventType, event.Op,
					baseURL, routePath)
			} else if event.Op == consts.CtrlWebContentAdd {
				var result string
				for _, content := range web.Contents {
					routePath := path.Join(web.Root, content.Path)
					result += fmt.Sprintf("[%s] %s: content add success, routePath is %s/%s\n",
						event.EventType, event.Op, baseURL, routePath)
				}
				return strings.TrimSuffix(result, "\n")
			}
			routePath := web.Root
			if !strings.HasPrefix(routePath, "/") {
				routePath = "/" + routePath
			}
			return fmt.Sprintf("[%s] %s: web %s on %s %d, routePath is %s%s", event.EventType, event.Op,
				pipeline.ListenerId, pipeline.Name, web.Port, baseURL, routePath)
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
	notifier    Notifier

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
	go broker.notifier.Send(&event)
}

func NewBroker() *eventBroker {
	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier: Notifier{notify: notify.New(),
			enable: false,
		},
		cache: NewMessageCache(eventBufSize),
		lock:  &sync.Mutex{},
	}
	go broker.Start()
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

type Notifier struct {
	notify *notify.Notify
	enable bool
}

func (broker *eventBroker) InitService(config *configs.NotifyConfig) error {
	if config == nil || !config.Enable {
		return nil
	}
	broker.notifier.enable = true
	if config.Telegram != nil && config.Telegram.Enable {
		tg, err := telegram.New(config.Telegram.APIKey)
		if err != nil {
			return err
		}
		tg.SetParseMode(telegram.ModeMarkdown)
		tg.AddReceivers(config.Telegram.ChatID)
		broker.notifier.notify.UseServices(tg)
	}
	if config.DingTalk != nil && config.DingTalk.Enable {
		dt := dingding.New(&dingding.Config{
			Token:  config.DingTalk.Token,
			Secret: config.DingTalk.Secret,
		})
		broker.notifier.notify.UseServices(dt)
	}
	if config.Lark != nil && config.Lark.Enable {
		lark := lark.NewWebhookService(config.Lark.WebHookUrl)
		broker.notifier.notify.UseServices(lark)
	}
	if config.ServerChan != nil && config.ServerChan.Enable {
		sc := http.New()
		sc.AddReceivers(&http.Webhook{
			URL:         config.ServerChan.URL,
			Method:      "POST",
			ContentType: "application/x-www-form-urlencoded",
			BuildPayload: func(subject, message string) (payload any) {
				data := url.Values{}
				data.Set("subject", subject)
				data.Set("message", message)
				return data.Encode()
			},
		})
		broker.notifier.notify.UseServices(sc)
	}
	return nil
}

func (n *Notifier) Send(event *Event) {
	if !n.enable {
		return
	}
	title := fmt.Sprintf("[%s] %s", event.EventType, event.Op)
	err := n.notify.Send(context.Background(), title, event.Message)
	if err != nil {
		logs.Log.Errorf("Failed to send notification: %s", err)
	}
	return
}
