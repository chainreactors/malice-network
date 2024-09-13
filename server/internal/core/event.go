package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/dingding"
	"github.com/nikoksr/notify/service/http"
	lark2 "github.com/nikoksr/notify/service/lark"
	"github.com/nikoksr/notify/service/telegram"
)

const (
	// Size is arbitrary, just want to avoid weird cases where we'd block on channel sends
	eventBufSize = 5
)

type Event struct {
	Session *Session
	Job     *Job
	Client  *Client
	Task    *Task

	EventType  string
	Op         string
	SourceName string
	Message    string
	Data       []byte
	Err        string
	IsNotify   bool
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
			logs.Log.Infof("[event.%s] %s: %s", event.EventType, event.Op, string(event.Data))
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
		err := broker.notifier.Send(&event)
		if err != nil {
			logs.Log.Errorf("Failed to send notification: %s", err)
			return
		}
	}
}

// Notify - Notify all third-patry services
func (broker *eventBroker) Notify(event Event) error {
	err := broker.notifier.Send(&event)
	if err != nil {
		return err
	}
	return nil
}

func newBroker() *eventBroker {
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
	return broker
}

var (
	// EventBroker - Distributes event messages
	EventBroker = newBroker()
)

type Notifier struct {
	notify *notify.Notify
	enable bool
}

func (broker *eventBroker) InitService(config *configs.NotifyConfig) error {
	if !config.Enable {
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
			Method:      config.ServerChan.Method,
			Header:      config.ServerChan.Headers,
			ContentType: config.ServerChan.ContentType,
			BuildPayload: func(subject, message string) (payload any) {
				return map[string]string{
					"subject": subject,
					"message": message,
				}
			},
		})
		broker.notifier.notify.UseServices(sc)
	}
	return nil
}

func (n *Notifier) Send(event *Event) error {
	if !n.enable {
		return nil
	}
	title := fmt.Sprintf("[%s] %s", event.EventType, event.Op)

	err := n.notify.Send(context.Background(), title, event.Message)
	if err != nil {
		return err
	}
	return nil
}
