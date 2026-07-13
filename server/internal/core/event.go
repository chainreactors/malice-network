package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	eventBufSize     = 25
	eventHistorySize = 512
)

var (
	ErrEventBrokerUnavailable = errors.New("event broker unavailable")
	ErrEventBrokerQueueFull   = errors.New("event broker queue full")
	ErrEventSubscriberSlow    = errors.New("event subscriber queue full")
	eventBrokerRestartBackoff = 200 * time.Millisecond
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
		sid := "unknown-session"
		if event.Session != nil && event.Session.SessionId != "" {
			sid = event.Session.SessionId
		}
		taskID := uint32(0)
		taskType := "unknown-task"
		if event.Task != nil {
			taskID = event.Task.TaskId
			if event.Task.Type != "" {
				taskType = event.Task.Type
			}
		}
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
				sid, taskID, taskType, event.Message)
		case consts.CtrlSessionError:
			return fmt.Sprintf("[%s] task: %d error: %s",
				sid, taskID, event.Err)
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
		if pipeline == nil {
			return fmt.Sprintf("[%s] %s: %s", event.EventType, event.Op, event.Message)
		}
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
						cont.Id, joinEventURL(baseURL, cont.Path))
				}
			} else if event.Op == consts.CtrlWebContentAdd {
				if cont := event.Job.FirstContent(); cont != nil {
					return fmt.Sprintf("[%s] %s: content add success, path: %s",
						event.EventType, event.Op, joinEventURL(baseURL, cont.Path))
				}
			}
			return kvView("web")
		case *clientpb.Pipeline_Custom:
			return kvView(pipeline.Type)
		}
	}
	return event.Message
}

func joinEventURL(baseURL, contentPath string) string {
	joined, err := url.JoinPath(baseURL, contentPath)
	if err == nil {
		return joined
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(contentPath, "/")
}

type Event struct {
	Session  *clientpb.Session
	Job      *clientpb.Job
	Client   *clientpb.Client
	Task     *clientpb.Task
	Spite    *implantpb.Spite
	Listener *clientpb.Listener

	Important bool
	EventType string
	Op        string
	Message   string
	Err       string
	Callee    string
	IsNotify  bool
}

// EventSubscription describes a resumable event consumer. StreamID and
// AfterSequence form the cursor returned by a previous subscription.
type EventSubscription struct {
	StreamID          string
	AfterSequence     uint64
	Replay            bool
	Topics            []string
	IncludeHeartbeats bool
}

// SequencedEvent is the broker-internal representation of an EventsV2
// envelope. Control envelopes have Ready or ResetRequired set and Event nil.
type SequencedEvent struct {
	StreamID       string
	Sequence       uint64
	OccurredAt     time.Time
	Event          *Event
	Replayed       bool
	ResetRequired  bool
	OldestSequence uint64
	LatestSequence uint64
	Ready          bool
}

func (event SequencedEvent) ToProtobuf() *clientpb.EventEnvelope {
	envelope := &clientpb.EventEnvelope{
		StreamId:       event.StreamID,
		Sequence:       event.Sequence,
		Replayed:       event.Replayed,
		ResetRequired:  event.ResetRequired,
		OldestSequence: event.OldestSequence,
		LatestSequence: event.LatestSequence,
		Ready:          event.Ready,
	}
	if !event.OccurredAt.IsZero() {
		envelope.OccurredAtUnixMilli = event.OccurredAt.UnixMilli()
	}
	if event.Event != nil {
		envelope.Event = event.Event.ToProtobuf()
	}
	return envelope
}

type eventV2Subscriber struct {
	channel           chan SequencedEvent
	topics            map[string]struct{}
	includeHeartbeats bool
	needsReset        bool
}

type eventV2SubscribeRequest struct {
	options EventSubscription
	channel chan SequencedEvent
	ready   chan struct{}
}

func (event *Event) String() string {
	var id string

	if event.Listener != nil {
		id = fmt.Sprintf("Listener %s", event.Listener.Id)
	} else if event.Job != nil {
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
		Listener:  event.Listener,
		Type:      event.EventType,
		Op:        event.Op,
		Formatted: event.format(),
		Message:   []byte(event.Message),
		Err:       event.Err,
		Callee:    event.Callee,
	}
}

type eventSubscription struct {
	events  chan Event
	ready   chan struct{}
	history chan []Event
}

type eventBroker struct {
	stop        chan struct{}
	publish     chan Event
	subscribe   chan eventSubscription
	unsubscribe chan chan Event
	send        chan Event
	notifier    inotify.Notifier

	cache *RingCache

	subscribeV2            chan eventV2SubscribeRequest
	unsubscribeV2          chan chan SequencedEvent
	v2Once                 sync.Once
	streamID               string
	sequence               uint64
	history                []SequencedEvent
	historyCapacity        int
	evictedThroughSequence uint64

	alive     atomic.Bool
	managed   atomic.Bool
	startOnce sync.Once
	stopOnce  sync.Once
	publishMu sync.Mutex
	stopped   bool
}

func (broker *eventBroker) run() error {
	broker.ensureV2()
	broker.alive.Store(true)
	subscribers := map[chan Event]struct{}{}
	v2Subscribers := map[chan SequencedEvent]*eventV2Subscriber{}
	defer func() {
		broker.alive.Store(false)
		for sub := range subscribers {
			closeEventSubscriber(sub)
		}
		for sub := range v2Subscribers {
			func(ch chan SequencedEvent) {
				defer func() {
					_ = recover()
				}()
				close(ch)
			}(sub)
		}
	}()
	for {
		select {
		case <-broker.stop:
			broker.recordAcceptedEvents()
			return nil
		case subscription := <-broker.subscribe:
			var history []Event
			if subscription.history != nil {
				for _, event := range broker.GetAll() {
					history = append(history, *event)
				}
			}
			subscribers[subscription.events] = struct{}{}
			if subscription.history != nil {
				subscription.history <- history
			}
			close(subscription.ready)
		case sub := <-broker.unsubscribe:
			delete(subscribers, sub)
		case request := <-broker.subscribeV2:
			subscriber := newEventV2Subscriber(request.channel, request.options)
			ready, replay := broker.prepareV2Subscription(request.options, subscriber)
			request.channel <- ready
			for _, event := range replay {
				request.channel <- event
			}
			v2Subscribers[request.channel] = subscriber
			close(request.ready)
		case sub := <-broker.unsubscribeV2:
			delete(v2Subscribers, sub)
		case event := <-broker.publish:
			sequenced := broker.recordAcceptedEvent(event)
			if event.Important {
				logs.Log.Infof("event.%s - %s", event.EventType, event.String())
			} else if event.EventType != consts.EventHeartbeat {
				logs.Log.Debugf("event.%s - %s", event.EventType, event.String())
			}
			for sub := range subscribers {
				if err := broker.dispatch(sub, event); err != nil {
					delete(subscribers, sub)
					closeEventSubscriber(sub)
					logs.Log.Warnf("disconnect event subscriber: %s", ErrorText(err))
				}
			}
			for channel, subscriber := range v2Subscribers {
				if err := broker.dispatchV2(subscriber, sequenced); err != nil {
					delete(v2Subscribers, channel)
					logs.Log.Warnf("drop broken EventsV2 subscriber: %s", ErrorText(err))
				}
			}
		}
	}
}

func (broker *eventBroker) recordAcceptedEvents() {
	for {
		select {
		case event := <-broker.publish:
			broker.recordAcceptedEvent(event)
		default:
			return
		}
	}
}

func closeEventSubscriber(sub chan Event) {
	defer func() {
		_ = recover()
	}()
	close(sub)
}

func (broker *eventBroker) recordAcceptedEvent(event Event) SequencedEvent {
	if event.Important {
		eventCopy := event
		broker.cache.Add(&eventCopy)
	}
	return broker.recordV2Event(event)
}

func (broker *eventBroker) ensureV2() {
	broker.v2Once.Do(func() {
		if broker.subscribeV2 == nil {
			broker.subscribeV2 = make(chan eventV2SubscribeRequest, eventBufSize)
		}
		if broker.unsubscribeV2 == nil {
			broker.unsubscribeV2 = make(chan chan SequencedEvent, eventBufSize)
		}
		if broker.historyCapacity <= 0 {
			broker.historyCapacity = eventHistorySize
		}
		if broker.streamID == "" {
			broker.streamID = newEventStreamID()
		}
	})
}

func newEventStreamID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func newEventV2Subscriber(channel chan SequencedEvent, options EventSubscription) *eventV2Subscriber {
	topics := make(map[string]struct{}, len(options.Topics))
	for _, topic := range options.Topics {
		if topic != "" {
			topics[topic] = struct{}{}
		}
	}
	return &eventV2Subscriber{
		channel:           channel,
		topics:            topics,
		includeHeartbeats: options.IncludeHeartbeats,
	}
}

func (broker *eventBroker) prepareV2Subscription(options EventSubscription, subscriber *eventV2Subscriber) (SequencedEvent, []SequencedEvent) {
	oldest, latest := broker.v2Bounds()
	resetRequired := false

	if options.StreamID != "" && options.StreamID != broker.streamID {
		resetRequired = true
	}
	if options.StreamID == "" && options.AfterSequence != 0 {
		resetRequired = true
	}
	if options.AfterSequence > latest {
		resetRequired = true
	}
	if options.Replay && options.StreamID != "" && options.AfterSequence < broker.evictedThroughSequence {
		resetRequired = true
	}

	ready := SequencedEvent{
		StreamID:       broker.streamID,
		Sequence:       latest,
		ResetRequired:  resetRequired,
		OldestSequence: oldest,
		LatestSequence: latest,
		Ready:          true,
	}
	if !options.Replay || resetRequired {
		return ready, nil
	}

	replay := make([]SequencedEvent, 0)
	for _, historical := range broker.history {
		if historical.Sequence <= options.AfterSequence || !subscriber.accepts(historical.Event) {
			continue
		}
		historical.Replayed = true
		historical.OldestSequence = oldest
		historical.LatestSequence = latest
		replay = append(replay, historical)
	}
	return ready, replay
}

func (broker *eventBroker) recordV2Event(event Event) SequencedEvent {
	broker.sequence++
	eventCopy := event
	sequenced := SequencedEvent{
		StreamID:   broker.streamID,
		Sequence:   broker.sequence,
		OccurredAt: time.Now(),
		Event:      &eventCopy,
	}
	if event.EventType == consts.EventHeartbeat || !event.Important {
		sequenced.OldestSequence, sequenced.LatestSequence = broker.v2Bounds()
		return sequenced
	}
	broker.history = append(broker.history, sequenced)
	if overflow := len(broker.history) - broker.historyCapacity; overflow > 0 {
		broker.evictedThroughSequence = broker.history[overflow-1].Sequence
		copy(broker.history, broker.history[overflow:])
		broker.history = broker.history[:broker.historyCapacity]
	}
	sequenced.OldestSequence, sequenced.LatestSequence = broker.v2Bounds()
	broker.history[len(broker.history)-1] = sequenced
	return sequenced
}

func (broker *eventBroker) v2Bounds() (uint64, uint64) {
	if len(broker.history) == 0 {
		return 0, broker.sequence
	}
	return broker.history[0].Sequence, broker.sequence
}

func (subscriber *eventV2Subscriber) accepts(event *Event) bool {
	if event == nil {
		return true
	}
	if event.EventType == consts.EventHeartbeat && !subscriber.includeHeartbeats {
		return false
	}
	if len(subscriber.topics) == 0 {
		return true
	}
	_, ok := subscriber.topics[event.EventType]
	return ok
}

func (broker *eventBroker) dispatchV2(subscriber *eventV2Subscriber, event SequencedEvent) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = RecoverError("event-v2-dispatch", recovered)
		}
	}()

	if subscriber.needsReset {
		oldest, latest := broker.v2Bounds()
		reset := SequencedEvent{
			StreamID:       broker.streamID,
			Sequence:       latest,
			ResetRequired:  true,
			OldestSequence: oldest,
			LatestSequence: latest,
		}
		select {
		case subscriber.channel <- reset:
			subscriber.needsReset = false
		default:
			return nil
		}
	}
	if !subscriber.accepts(event.Event) {
		return nil
	}
	select {
	case subscriber.channel <- event:
	default:
		if event.Event == nil || event.Event.Important {
			subscriber.needsReset = true
		}
	}
	return nil
}

func (broker *eventBroker) dispatch(sub chan Event, event Event) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = RecoverError("event-dispatch", recovered)
		}
	}()
	select {
	case sub <- event:
		return nil
	default:
		// Preserve legacy V1 behavior by dropping the current event without
		// disconnecting the stream. V2 consumers receive reset/replay signals.
		return nil
	}
}

func (broker *eventBroker) Start() {
	broker.ensureV2()
	broker.startOnce.Do(func() {
		broker.managed.Store(true)
		go func() {
			for {
				err := RunGuarded("event-broker", broker.run, LogGuardedError("event-broker"))
				if err == nil {
					return
				}
				select {
				case <-broker.stop:
					return
				case <-time.After(eventBrokerRestartBackoff):
				}
			}
		}()
	})
}

// Stop - Close the broker channel
func (broker *eventBroker) Stop() {
	if broker == nil {
		return
	}
	broker.stopOnce.Do(func() {
		broker.publishMu.Lock()
		broker.stopped = true
		close(broker.stop)
		broker.publishMu.Unlock()
	})
}

// Subscribe - Generate a new subscription channel
func (broker *eventBroker) Subscribe() (chan Event, error) {
	events, _, err := broker.subscribeEvents(false)
	return events, err
}

func (broker *eventBroker) SubscribeWithHistory() (chan Event, []Event, error) {
	return broker.subscribeEvents(true)
}

func (broker *eventBroker) subscribeEvents(includeHistory bool) (chan Event, []Event, error) {
	events := make(chan Event, eventBufSize)
	ready := make(chan struct{})
	var history chan []Event
	if includeHistory {
		history = make(chan []Event, 1)
	}
	select {
	case broker.subscribe <- eventSubscription{events: events, ready: ready, history: history}:
	case <-broker.stop:
		return nil, nil, ErrEventBrokerUnavailable
	}
	select {
	case <-ready:
		if history != nil {
			return events, <-history, nil
		}
		return events, nil, nil
	case <-broker.stop:
		return nil, nil, ErrEventBrokerUnavailable
	}
}

// Unsubscribe - Remove a subscription channel
func (broker *eventBroker) Unsubscribe(events chan Event) {
	select {
	case broker.unsubscribe <- events:
	case <-broker.stop:
	}
	//close(events)
}

// SubscribeV2 registers a resumable event subscription. The first value in
// the returned channel is always a Ready envelope containing stream bounds.
func (broker *eventBroker) SubscribeV2(options EventSubscription) (chan SequencedEvent, error) {
	if broker == nil {
		return nil, ErrEventBrokerUnavailable
	}
	broker.ensureV2()
	capacity := broker.historyCapacity + 1
	if capacity < eventBufSize {
		capacity = eventBufSize
	}
	events := make(chan SequencedEvent, capacity)
	ready := make(chan struct{})
	request := eventV2SubscribeRequest{
		options: options,
		channel: events,
		ready:   ready,
	}
	select {
	case broker.subscribeV2 <- request:
	case <-broker.stop:
		return nil, ErrEventBrokerUnavailable
	}
	select {
	case <-ready:
		return events, nil
	case <-broker.stop:
		return nil, ErrEventBrokerUnavailable
	}
}

func (broker *eventBroker) UnsubscribeV2(events chan SequencedEvent) {
	if broker == nil {
		return
	}
	broker.ensureV2()
	select {
	case broker.unsubscribeV2 <- events:
	case <-broker.stop:
	}
}

// Publish - Push a message to all subscribers
func (broker *eventBroker) Publish(event Event) {
	if broker == nil {
		return
	}
	if err := broker.TryPublish(event); err != nil && event.EventType != consts.EventHeartbeat {
		logs.Log.Errorf("event publish failed [%s.%s]: %s", event.EventType, event.Op, err)
	}
}

func (broker *eventBroker) TryPublish(event Event) error {
	if broker == nil {
		return ErrEventBrokerUnavailable
	}
	broker.publishMu.Lock()
	if broker.stopped {
		broker.publishMu.Unlock()
		return ErrEventBrokerUnavailable
	}
	if broker.managed.Load() && !broker.alive.Load() {
		broker.publishMu.Unlock()
		return ErrEventBrokerUnavailable
	}
	select {
	case broker.publish <- event:
	default:
		broker.publishMu.Unlock()
		return ErrEventBrokerQueueFull
	}
	broker.publishMu.Unlock()
	if event.IsNotify {
		broker.Notify(event)
	}
	return nil
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
	if broker == nil {
		return
	}
	GoGuarded("event-notify", func() error {
		broker.notifier.Send(event.EventType, event.Op, event.Message)
		return nil
	}, LogGuardedError("event-notify"))
}

func NewBroker() *eventBroker {
	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize),
		subscribe:   make(chan eventSubscription),
		unsubscribe: make(chan chan Event, eventBufSize),
		send:        make(chan Event, eventBufSize),
		notifier:    inotify.NewNotifier(),
		cache:       NewMessageCache(eventBufSize),
	}
	broker.Start()
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
