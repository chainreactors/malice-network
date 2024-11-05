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
	lark2 "github.com/nikoksr/notify/service/lark"
	"github.com/nikoksr/notify/service/telegram"
	"net/url"
)

const (
	// Size is arbitrary, just want to avoid weird cases where we'd block on channel sends
	eventBufSize = 5
)

type Event struct {
	Session *clientpb.Session
	Job     *clientpb.Job
	Client  *clientpb.Client
	Task    *clientpb.Task
	Spite   *implantpb.Spite

	EventType string
	Op        string
	Message   string
	Err       string
	Callee    string
	IsNotify  bool
}

type eventBroker struct {
	stop        chan struct{}
	publish     chan Event
	subscribe   chan chan Event
	unsubscribe chan chan Event
	send        chan Event
	notifier    Notifier
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
			if event.EventType != consts.EventHeartbeat {
				logs.Log.Infof("[event.%s] %s: %s", event.EventType, event.Op, event.Message)
			}

			for sub := range subscribers {
				sub <- event
			}
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
	close(events)
}

// Publish - Push a message to all subscribers
func (broker *eventBroker) Publish(event Event) {
	broker.publish <- event
	if event.IsNotify {
		broker.Notify(event)
	}
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
			enable: false},
	}
	go broker.Start()
	ticker := NewTicker()

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
	if config.Telegram.Enable {
		tg, err := telegram.New(config.Telegram.APIKey)
		if err != nil {
			return err
		}
		tg.SetParseMode(telegram.ModeMarkdown)
		tg.AddReceivers(config.Telegram.ChatID)
		broker.notifier.notify.UseServices(tg)
	}
	if config.DingTalk.Enable {
		dt := dingding.New(&dingding.Config{
			Token:  config.DingTalk.Token,
			Secret: config.DingTalk.Secret,
		})
		broker.notifier.notify.UseServices(dt)
	}
	if config.Lark.Enable {
		lark := lark2.NewWebhookService(config.Lark.WebHookUrl)
		broker.notifier.notify.UseServices(lark)
	}
	if config.ServerChan.Enable {
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
