package core

import (
	"context"
	"fmt"
	"io"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/helper/utils/output"
)

func (s *ServerStatus) AddDoneCallback(task *clientpb.Task, callback TaskCallback) {
	s.doneCallbacks.Store(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId), callback)
}

func (s *ServerStatus) AddCallback(task *clientpb.Task, callback TaskCallback) {
	s.finishCallbacks.Store(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId), callback)
}

func (s *ServerStatus) triggerTaskDone(event *clientpb.Event) {
	task := event.GetTask()
	var sess *Session
	var ok bool
	var err error
	sess, ok = s.GetLocalSession(event.Task.SessionId)
	if !ok {
		sess, err = s.UpdateSession(event.Task.SessionId)
		if err != nil {
			Log.Errorf("session not found: %s\n", event.Task.SessionId)
			return
		}
	}

	log := s.ObserverLog(event.Task.SessionId)
	err = handler.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()) + "\n")
		return
	}
	HandlerTask(sess, &clientpb.TaskContext{
		Task:    event.Task,
		Session: event.Session,
		Spite:   event.Spite,
	}, event.Message, event.Callee, false)

	if callback, ok := s.finishCallbacks.Load(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId)); ok {
		callback.(TaskCallback)(event.Spite)
	}
}

func (s *ServerStatus) triggerTaskFinish(event *clientpb.Event) {
	task := event.GetTask()
	var sess *Session
	var ok bool
	var err error
	sess, ok = s.GetLocalSession(event.Task.SessionId)
	if !ok {
		sess, err = s.UpdateSession(event.Task.SessionId)
		if err != nil {
			Log.Errorf("session not found: %s\n", event.Task.SessionId)
			return
		}
	}

	log := s.ObserverLog(event.Task.SessionId)
	err = handler.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()) + "\n")
		return
	}

	HandlerTask(sess, &clientpb.TaskContext{
		Task:    event.Task,
		Session: event.Session,
		Spite:   event.Spite,
	}, event.Message, event.Callee, true)

	callbackId := fmt.Sprintf("%s_%d", task.SessionId, task.TaskId)
	if callback, ok := s.finishCallbacks.Load(callbackId); ok {
		callback.(TaskCallback)(event.Spite)
		s.finishCallbacks.Delete(callbackId)
		s.doneCallbacks.Delete(callbackId)
	}
}

func HandlerTask(sess *Session, ctx *clientpb.TaskContext, message []byte, callee string, isFinish bool) {
	log := sess.Log
	var callback intermediate.ImplantCallback
	fn, ok := intermediate.InternalFunctions[ctx.Task.Type]
	if !ok {
		log.Debugf("function %s not found\n", ctx.Task.Type)
		status, err := output.ParseStatus(ctx)
		if err != nil {
			log.Importantf("parse status error: %s\n", err)
			return
		}
		log.Importantf("task %d %t", ctx.Task.TaskId, status)
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

	s := logs.GreenBold(fmt.Sprintf("[%s.%d] %s (%d/%d),%s\n",
		ctx.Task.SessionId, ctx.Task.TaskId, prompt,
		ctx.Task.Cur, ctx.Task.Total,
		message))
	log.Importantf(s)
	if callee != consts.CalleeCMD {
		return
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
		log.Errorf(logs.RedBold(err.Error()))
	} else {
		log.Console(resp + "\n")
	}
}

func (s *ServerStatus) AddEventHook(event intermediate.EventCondition, callback intermediate.OnEventFunc) {
	if _, ok := s.EventHook[event]; !ok {
		s.EventHook[event] = []intermediate.OnEventFunc{}
	}
	s.EventHook[event] = append(s.EventHook[event], callback)
}

func (s *ServerStatus) EventHandler() {
	eventStream, err := s.Rpc.Events(context.Background(), &clientpb.Empty{})
	if err != nil {
		return
	}
	s.Update()
	s.EventStatus = true
	Log.Debugf("starting event loop\n")
	defer func() {
		Log.Warnf("event stream broken\n")
		s.EventStatus = false
	}()
	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			return
		}
		for condition, fns := range s.EventHook {
			if condition.Match(event) {
				go func() {
					for _, fn := range fns {
						_, err := fn(event)
						if err != nil {
							Log.Errorf("error running event hook: %s", err)
						}
					}
				}()
			}
		}
		go func() {
			s.handlerEvent(event)
		}()

	}
}

func (s *ServerStatus) handlerEvent(event *clientpb.Event) {
	switch event.Type {
	case consts.EventClient:
		if event.Op == consts.CtrlClientJoin {
			Log.Info(event.Formatted + "\n")
		} else if event.Op == consts.CtrlClientLeft {
			Log.Info(event.Formatted + "\n")
		}
	case consts.EventBroadcast:
		Log.Info(event.Formatted + "\n")
	case consts.EventSession:
		s.handlerSession(event)
	case consts.EventNotify:
		Log.Important(event.Formatted + "\n")
	case consts.EventJob:
		s.handleJob(event)
	case consts.EventListener:
		Log.Important(event.Formatted + "\n")
	case consts.EventTask:
		s.handlerTask(event)
	case consts.EventWebsite:
		Log.Important(event.Formatted + "\n")
	case consts.EventBuild:
		Log.Important(event.Formatted + "\n")
	case consts.EventPivot:
		Log.Important(event.Formatted + "\n")
	case consts.EventContext:
		Log.Important(event.Formatted + "\n")
	}
}

func (s *ServerStatus) handleJob(event *clientpb.Event) {
	if event.Err != "" {
		Log.Errorf("[%s] %s: %s\n", event.Type, event.Op, event.Err)
		return
	}
	pipeline := event.GetJob().GetPipeline()
	if event.Op == consts.CtrlPipelineSync {
		s.Pipelines[pipeline.Name] = pipeline
	} else {
		Log.Important(event.Formatted + "\n")
	}
}

func (s *ServerStatus) handlerTask(event *clientpb.Event) {
	switch event.Op {
	case consts.CtrlTaskCallback:
		s.triggerTaskDone(event)
	case consts.CtrlTaskFinish:
		s.triggerTaskFinish(event)
	case consts.CtrlTaskCancel:
		Log.Importantf("[%s.%d] task canceled\n", event.Task.SessionId, event.Task.TaskId)
	case consts.CtrlTaskError:
		Log.Errorf("[%s.%d] %s\n", event.Task.SessionId, event.Task.TaskId, event.Err)
	}
}

func (s *ServerStatus) handlerSession(event *clientpb.Event) {
	sid := event.Session.SessionId
	switch event.Op {
	case consts.CtrlSessionRegister:
		s.AddSession(event.Session)
		Log.Important(event.Formatted + "\n")
	case consts.CtrlSessionTask:
		Log.Info(event.Formatted + "\n")
	case consts.CtrlSessionError:
		log := s.ObserverLog(sid)
		log.Error(event.Formatted + "\n")
	case consts.CtrlSessionLog:
		log := s.ObserverLog(sid)
		log.Error(event.Formatted + "\n")
	case consts.CtrlSessionLeave:
		Log.Important(event.Formatted + "\n")
	case consts.CtrlSessionReborn:
		s.AddSession(event.Session)
		Log.Important(event.Formatted + "\n")
	}
}
