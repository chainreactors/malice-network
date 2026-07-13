package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var ErrLuaVMDead = fmt.Errorf("lua vm is dead")

const eventStreamHeaderTimeout = 10 * time.Second

func wrapToTaskContext(event *clientpb.Event) *clientpb.TaskContext {
	return &clientpb.TaskContext{
		Task:    event.Task,
		Session: event.Session,
		Spite:   event.Spite,
	}
}

type Server struct {
	*client.ServerState
	taskMessageMu         sync.Mutex
	taskMessages          map[string]string
	eventHookMu           sync.RWMutex
	eventHookIDs          map[client.EventCondition][]uint64
	nextEventHookID       uint64
	eventHandlerMu        sync.Mutex
	eventHandlerActive    bool
	suppressStartupOutput bool
	startupHistoryPending bool

	// Quiet suppresses console event output while still updating internal state.
	Quiet bool
}

func (s *Server) claimEventHandler() bool {
	if s == nil {
		return false
	}
	s.eventHandlerMu.Lock()
	defer s.eventHandlerMu.Unlock()
	if s.eventHandlerActive {
		return false
	}
	s.eventHandlerActive = true
	s.EventStatus = true
	return true
}

func (s *Server) releaseEventHandler() {
	if s == nil {
		return
	}
	s.eventHandlerMu.Lock()
	defer s.eventHandlerMu.Unlock()
	s.eventHandlerActive = false
	s.EventStatus = false
}

func (s *Server) EventHandlerRunning() bool {
	if s == nil {
		return false
	}
	s.eventHandlerMu.Lock()
	defer s.eventHandlerMu.Unlock()
	return s.eventHandlerActive
}

func (s *Server) needsStartupHistory() bool {
	s.eventHandlerMu.Lock()
	defer s.eventHandlerMu.Unlock()
	return s.startupHistoryPending
}

func (s *Server) markStartupHistoryComplete() {
	s.eventHandlerMu.Lock()
	s.startupHistoryPending = false
	s.eventHandlerMu.Unlock()
}

func taskMessageKey(sessionID string, taskID uint32) string {
	return fmt.Sprintf("%s-%d", sessionID, taskID)
}

func (s *Server) appendTaskMessage(task *clientpb.Task, message []byte) {
	if s == nil || task == nil || len(message) == 0 {
		return
	}

	msg := string(message)
	if msg == "" {
		return
	}

	key := taskMessageKey(task.SessionId, task.TaskId)
	s.taskMessageMu.Lock()
	defer s.taskMessageMu.Unlock()
	if prev, ok := s.taskMessages[key]; ok && prev != "" {
		s.taskMessages[key] = prev + "\n" + msg
		return
	}
	s.taskMessages[key] = msg
}

func (s *Server) popTaskMessage(sessionID string, taskID uint32) string {
	if s == nil {
		return ""
	}

	key := taskMessageKey(sessionID, taskID)
	s.taskMessageMu.Lock()
	defer s.taskMessageMu.Unlock()
	msg := s.taskMessages[key]
	delete(s.taskMessages, key)
	return msg
}

// NewServer wraps client.ServerState into core.ServerState
func NewServer(conn *grpc.ClientConn, config *mtls.ClientConfig) (*Server, error) {
	return NewServerWithOptions(conn, config, false)
}

func NewServerWithOptions(conn *grpc.ClientConn, config *mtls.ClientConfig, suppressStartupOutput bool) (*Server, error) {
	s, err := client.NewServerStatus(conn, config)
	if err != nil {
		return nil, err
	}
	ser := &Server{
		ServerState:           s,
		taskMessages:          make(map[string]string),
		suppressStartupOutput: suppressStartupOutput,
		startupHistoryPending: true,
	}
	return ser, nil
}

func (s *Server) replayCachedEvents(events []*clientpb.Event, suppressOutput bool) {
	if suppressOutput {
		return
	}
	for _, event := range events {
		if event == nil || event.Type == consts.EventTask {
			continue
		}
		s.handleEventOutput(event)
	}
}

func (s *Server) AddDoneCallback(task *clientpb.Task, callback client.TaskCallback) {
	s.DoneCallbacks.Store(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId), callback)
}

func (s *Server) AddCallback(task *clientpb.Task, callback client.TaskCallback) {
	s.FinishCallbacks.Store(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId), callback)
}

func (s *Server) triggerTaskDone(event *clientpb.Event) {
	if s == nil || event == nil || event.Task == nil {
		return
	}
	task := event.GetTask()
	sess, err := s.GetOrUpdateSession(event.Task.SessionId)
	if err != nil {
		client.Log.Errorf("session not found: %s\n", event.Task.SessionId)
		return
	}

	log := s.ObserverLog(event.Task.SessionId)
	err = types.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf("%s\n", logs.RedBold(err.Error()))
		return
	}
	taskContext := wrapToTaskContext(event)
	s.appendTaskMessage(task, event.Message)
	if event.Callee != consts.CalleeRPC {
		HandlerTask(sess, log, taskContext, event.Message, event.Callee, false)
	}

	if callback, ok := s.DoneCallbacks.Load(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId)); ok {
		callback.(client.TaskCallback)(taskContext)
	}
}

func (s *Server) triggerTaskFinish(event *clientpb.Event) {
	if s == nil || event == nil || event.Task == nil {
		return
	}
	task := event.GetTask()
	sess, err := s.GetOrUpdateSession(event.Task.SessionId)
	if err != nil {
		client.Log.Errorf("session not found: %s\n", event.Task.SessionId)
		return
	}

	log := s.ObserverLog(event.Task.SessionId)
	err = types.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf("%s\n", logs.RedBold(err.Error()))
		return
	}
	taskContext := wrapToTaskContext(event)
	s.appendTaskMessage(task, event.Message)
	if event.Callee != consts.CalleeRPC {
		HandlerTask(sess, log, taskContext, event.Message, event.Callee, true)
	}

	callbackId := fmt.Sprintf("%s-%d", task.SessionId, task.TaskId)
	if callback, ok := s.FinishCallbacks.Load(callbackId); ok {
		callback.(client.TaskCallback)(taskContext)
		s.FinishCallbacks.Delete(callbackId)
		s.DoneCallbacks.Delete(callbackId)
	}
}

func HandlerTask(sess *client.Session, log *client.Logger, ctx *clientpb.TaskContext, message []byte, callee string, isFinish bool) {
	sess.Locker.Lock()
	defer sess.Locker.Unlock()
	var callback intermediate.ImplantCallback
	fn, ok := intermediate.InternalFunctions[ctx.Task.Type]
	if !ok {
		log.Debugf("function %s not found\n", ctx.Task.Type)
		status, err := output.ParseStatus(ctx)
		if err != nil {
			log.Importantf("parse status error: %s\n", err)
		} else {
			log.Importantf("%s\n", status)
		}
		return
	}
	var prompt string
	if isFinish {
		prompt = "task finish"
		if fn.FinishCallback == nil {
			log.Consolef("%s not impl output impl\n", ctx.Task.Type)
			return
		}
		callback = fn.FinishCallback
	} else {
		prompt = "task done"
		if fn.DoneCallback == nil {
			log.Debugf("%s not impl output impl\n", ctx.Task.Type)
			return
		}
		callback = fn.DoneCallback
	}

	s := logs.GreenBold(fmt.Sprintf("[%s.%d] %s (%s),%s\n",
		ctx.Task.SessionId, ctx.Task.TaskId, prompt,
		ctx.Task.Progress(),
		message))

	if callee != consts.CalleePty && callee != consts.CalleeGui {
		log.Importantf("%s", s)
	}

	var err error
	var resp string

	if isFinish {
		log.FileLog(s)
		resp, err = callback(ctx)
		log.FileLog(resp + "\n")
	} else {
		resp, err = callback(ctx)
	}

	if err != nil {
		log.Errorf("%s", logs.RedBold(err.Error()))
		return
	}

	if resp != "" && (callee == consts.CalleeCMD || callee == consts.CalleeMal || callee == consts.CalleeRPC || callee == consts.CalleeMCP) {
		if log != client.MuteLog {
			asyncPrint("%s\n", resp)
		}
	}
}

func (s *Server) AddEventHook(event client.EventCondition, callback client.OnEventFunc) uint64 {
	if s == nil {
		return 0
	}
	s.eventHookMu.Lock()
	defer s.eventHookMu.Unlock()
	if s.EventHook == nil {
		s.EventHook = map[client.EventCondition][]client.OnEventFunc{}
	}
	if _, ok := s.EventHook[event]; !ok {
		s.EventHook[event] = []client.OnEventFunc{}
	}
	if s.eventHookIDs == nil {
		s.eventHookIDs = map[client.EventCondition][]uint64{}
	}
	for len(s.eventHookIDs[event]) < len(s.EventHook[event]) {
		s.eventHookIDs[event] = append(s.eventHookIDs[event], 0)
	}
	s.nextEventHookID++
	s.EventHook[event] = append(s.EventHook[event], callback)
	s.eventHookIDs[event] = append(s.eventHookIDs[event], s.nextEventHookID)
	return s.nextEventHookID
}

type eventHookGroup struct {
	condition client.EventCondition
	hooks     []client.OnEventFunc
	ids       []uint64
}

func (s *Server) matchingEventHooks(event *clientpb.Event) []eventHookGroup {
	if s == nil || event == nil {
		return nil
	}
	s.eventHookMu.RLock()
	defer s.eventHookMu.RUnlock()

	if len(s.EventHook) == 0 {
		return nil
	}

	groups := make([]eventHookGroup, 0, len(s.EventHook))
	for condition, hooks := range s.EventHook {
		conditionCopy := condition
		if !conditionCopy.Match(event) {
			continue
		}
		hooksCopy := append([]client.OnEventFunc(nil), hooks...)
		idsCopy := append([]uint64(nil), s.eventHookIDs[condition]...)
		groups = append(groups, eventHookGroup{
			condition: conditionCopy,
			hooks:     hooksCopy,
			ids:       idsCopy,
		})
	}
	return groups
}

func (s *Server) RemoveEventHook(condition client.EventCondition, target client.OnEventFunc) {
	if s == nil || target == nil {
		return
	}

	s.eventHookMu.Lock()
	defer s.eventHookMu.Unlock()

	hooks, ok := s.EventHook[condition]
	if !ok {
		return
	}

	targetPtr := reflect.ValueOf(target).Pointer()
	filtered := make([]client.OnEventFunc, 0, len(hooks))
	filteredIDs := make([]uint64, 0, len(hooks))
	ids := s.eventHookIDs[condition]
	for index, hook := range hooks {
		if hook == nil {
			continue
		}
		if reflect.ValueOf(hook).Pointer() == targetPtr {
			continue
		}
		filtered = append(filtered, hook)
		if index < len(ids) {
			filteredIDs = append(filteredIDs, ids[index])
		}
	}
	if len(filtered) == 0 {
		delete(s.EventHook, condition)
		delete(s.eventHookIDs, condition)
		return
	}
	s.EventHook[condition] = filtered
	s.eventHookIDs[condition] = filteredIDs
}

func (s *Server) RemoveEventHookByID(condition client.EventCondition, targetID uint64) {
	if s == nil || targetID == 0 {
		return
	}
	s.eventHookMu.Lock()
	defer s.eventHookMu.Unlock()

	hooks := s.EventHook[condition]
	ids := s.eventHookIDs[condition]
	filteredHooks := make([]client.OnEventFunc, 0, len(hooks))
	filteredIDs := make([]uint64, 0, len(ids))
	for index, hook := range hooks {
		if index < len(ids) && ids[index] == targetID {
			continue
		}
		filteredHooks = append(filteredHooks, hook)
		if index < len(ids) {
			filteredIDs = append(filteredIDs, ids[index])
		}
	}
	if len(filteredHooks) == 0 {
		delete(s.EventHook, condition)
		delete(s.eventHookIDs, condition)
		return
	}
	s.EventHook[condition] = filteredHooks
	s.eventHookIDs[condition] = filteredIDs
}

func (s *Server) dispatchEventHooks(event *clientpb.Event) {
	if s == nil || event == nil {
		return
	}

	for _, group := range s.matchingEventHooks(event) {
		condition := group.condition
		hooks := group.hooks
		ids := group.ids
		go func(condition client.EventCondition, hooks []client.OnEventFunc, ids []uint64) {
			for index, hook := range hooks {
				if hook == nil {
					continue
				}
				_, err := hook(event)
				if err != nil {
					if errors.Is(err, ErrLuaVMDead) {
						if index < len(ids) {
							s.RemoveEventHookByID(condition, ids[index])
						} else {
							s.RemoveEventHook(condition, hook)
						}
					} else {
						client.Log.Errorf("error running event hook: %s", err)
					}
				}
			}
		}(condition, hooks, ids)
	}
}

func (s *Server) EventHandler() {
	if !s.claimEventHandler() {
		return
	}
	defer s.releaseEventHandler()

	replayStartupHistory := s.needsStartupHistory()
	eventBaseContext, cancelEventStream := context.WithCancel(context.Background())
	defer cancelEventStream()
	eventContext := eventBaseContext
	if replayStartupHistory {
		eventContext = metadata.NewOutgoingContext(eventContext, metadata.Pairs(
			consts.EventStreamHistoryReplayHeader, "true",
		))
	}
	eventStream, err := s.Rpc.Events(eventContext, &clientpb.Empty{})
	if err != nil {
		return
	}
	type headerResult struct {
		header metadata.MD
		err    error
	}
	headerCh := make(chan headerResult, 1)
	go func() {
		header, headerErr := eventStream.Header()
		headerCh <- headerResult{header: header, err: headerErr}
	}()
	headerTimer := time.NewTimer(eventStreamHeaderTimeout)
	defer headerTimer.Stop()
	var header metadata.MD
	select {
	case result := <-headerCh:
		if result.err != nil {
			return
		}
		header = result.header
	case <-headerTimer.C:
		client.Log.Warnf("event stream header timed out after %s\n", eventStreamHeaderTimeout)
		return
	}
	if replayStartupHistory {
		historyCount := 0
		if values := header.Get(consts.EventStreamHistoryCountHeader); len(values) > 0 {
			parsed, parseErr := strconv.Atoi(values[0])
			if parseErr != nil || parsed < 0 {
				client.Log.Warnf("invalid event history count %q\n", values[0])
				return
			}
			historyCount = parsed
		} else {
			// Older servers cannot provide an atomic history/live cut. Fetching
			// history after subscription avoids loss but may replay a concurrent
			// important event once; this is an intentional compatibility fallback.
			historyContext, cancelHistory := context.WithTimeout(eventBaseContext, eventStreamHeaderTimeout)
			history, historyErr := s.GetEvent(historyContext, &clientpb.Int{})
			cancelHistory()
			if historyErr != nil {
				client.Log.Warnf("failed to load event history: %v\n", historyErr)
				return
			}
			s.replayCachedEvents(history.GetEvents(), s.suppressStartupOutput)
		}
		for i := 0; i < historyCount; i++ {
			event, recvErr := eventStream.Recv()
			if recvErr != nil {
				client.Log.Warnf("event history receive failed: %v\n", recvErr)
				return
			}
			if !s.suppressStartupOutput && event != nil && event.Type != consts.EventTask {
				s.handleEventOutput(event)
			}
		}
		s.markStartupHistoryComplete()
	}
	if err := s.Update(); err != nil {
		client.Log.Warnf("failed to refresh server state: %v\n", err)
	}
	if session := s.Get(); session != nil {
		if _, err := s.UpdateSession(session.SessionId); err != nil {
			client.Log.Warnf("failed to refresh active session %s: %v\n", session.SessionId, err)
		}
	}
	client.Log.Info("starting event loop\n")
	defer func() {
		client.Log.Warnf("event stream broken\n")
	}()
	for {
		event, err := eventStream.Recv()
		if err != nil {
			if err != io.EOF {
				client.Log.Warnf("event stream receive failed: %v\n", err)
			}
			return
		}
		if event == nil {
			return
		}
		s.dispatchEventHooks(event)

		if fn, ok := s.GetEventCallback(event.Op); ok {
			fn(event)
		}
		s.HandlerEvent(event)
	}
}

// renderEvent applies CLI-specific coloring to plain event text based on event type/op.
// Server sends plain text in Formatted; coloring is the client's responsibility.
func renderEvent(event *clientpb.Event) string {
	if event == nil {
		return ""
	}
	switch event.Type {
	case consts.EventSession:
		switch event.Op {
		case consts.CtrlSessionRegister, consts.CtrlSessionReborn, consts.CtrlSessionInit:
			return logs.GreenBold(event.Formatted)
		case consts.CtrlSessionDead:
			return logs.YellowBold(event.Formatted)
		case consts.CtrlSessionError:
			return logs.RedBold(event.Formatted)
		case consts.CtrlSessionTask:
			return logs.GreenBold(event.Formatted)
		}
	case consts.EventJob:
		if event.Err != "" {
			return logs.RedBold(event.Formatted)
		}
	case consts.EventListener:
		switch event.Op {
		case consts.CtrlListenerStart:
			return logs.GreenBold(event.Formatted)
		case consts.CtrlListenerStop:
			return logs.YellowBold(event.Formatted)
		}
	}
	return event.Formatted
}

func (s *Server) HandlerEvent(event *clientpb.Event) {
	if s == nil || event == nil {
		return
	}
	// Reconcile state first — single entry point for all map updates
	s.ReconcileEvent(event)
	s.handleEventOutput(event)
}
func (s *Server) handleEventOutput(event *clientpb.Event) {
	if s == nil || event == nil {
		return
	}
	// Quiet mode (non-index mux pane): suppress notification events but let
	// task events through so user-initiated commands still show results.
	if s.Quiet && event.Type != consts.EventTask {
		return
	}

	// Then handle UI/logging
	switch event.Type {
	case consts.EventClient:
		if event.Op == consts.CtrlClientJoin || event.Op == consts.CtrlClientLeft {
			client.Log.Info(event.Formatted + "\n")
		}
	case consts.EventBroadcast:
		client.Log.Info(event.Formatted + "\n")
	case consts.EventSession:
		s.handlerSession(event)
	case consts.EventNotify:
		client.Log.Important(event.Formatted + "\n")
	case consts.EventJob:
		s.handleJob(event)
	case consts.EventListener:
		colored := renderEvent(event)
		client.Log.Important(colored + "\n")
	case consts.EventTask:
		s.handlerTask(event)
	case consts.EventWebsite:
		client.Log.Important(event.Formatted + "\n")
	case consts.EventBuild:
		client.Log.Important(event.Formatted + "\n")
	case consts.EventPivot:
		client.Log.Important(event.Formatted + "\n")
	case consts.EventContext:
		client.Log.Important(event.Formatted + "\n")
	case consts.EventCert:
		client.Log.Important(event.Formatted + "\n")
	}
}

func (s *Server) handleJob(event *clientpb.Event) {
	if event == nil {
		return
	}
	if event.Err != "" {
		client.Log.Errorf("[%s] %s: %s\n", event.Type, event.Op, event.Err)
		return
	}
	// State updates are handled by ReconcileEvent; here we only log.
	colored := renderEvent(event)
	switch event.Op {
	case consts.CtrlPipelineSync, consts.CtrlRemAgentReconfigure:
		// silent sync/reconfigure, no log
	default:
		client.Log.Important(colored + "\n")
	}
}

func (s *Server) handlerTask(event *clientpb.Event) {
	if s == nil || event == nil || event.Task == nil {
		return
	}
	switch event.Op {
	case consts.CtrlTaskCallback:
		s.triggerTaskDone(event)
	case consts.CtrlTaskFinish:
		s.triggerTaskFinish(event)
	case consts.CtrlTaskCancel:
		if event.Callee != consts.CalleeGui {
			log := s.ObserverLog(event.Task.SessionId)
			log.Importantf("[%s.%d] task canceled\n", event.Task.SessionId, event.Task.TaskId)
		}
	case consts.CtrlTaskError:
		if event.Callee != consts.CalleeGui {
			log := s.ObserverLog(event.Task.SessionId)
			log.Errorf("[%s.%d] %s\n", event.Task.SessionId, event.Task.TaskId, event.Err)
		}
	}
}

func (s *Server) handlerSession(event *clientpb.Event) {
	if s == nil || event == nil || event.Session == nil {
		return
	}
	// State updates are handled by ReconcileEvent; here we only handle UI/logging.
	sid := event.Session.SessionId
	colored := renderEvent(event)
	switch event.Op {
	case consts.CtrlSessionRegister:
		client.Log.Important(colored + "\n")
	case consts.CtrlSessionUpdate:
		// silent update, no log
	case consts.CtrlSessionTask:
		log := s.ObserverLog(sid)
		log.Info(colored + "\n")
	case consts.CtrlSessionError:
		log := s.ObserverLog(sid)
		log.Error(colored + "\n")
	case consts.CtrlSessionLog:
		log := s.ObserverLog(sid)
		log.Error(event.Formatted + "\n")
	case consts.CtrlSessionDead:
		client.Log.Important(colored + "\n")
	case consts.CtrlSessionInit:
		client.Log.Important(colored + "\n")
	case consts.CtrlSessionReborn:
		client.Log.Important(colored + "\n")
	}
}
