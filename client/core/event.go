package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"io"
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

func HandlerTask(sess *Session, context *clientpb.TaskContext, message []byte, callee string, isFinish bool) {
	log := sess.Log
	var callback intermediate.ImplantCallback
	fn, ok := intermediate.InternalFunctions[context.Task.Type]
	if !ok {
		log.Errorf("function %s not found\n", context.Task.Type)
		return
	}
	var prompt string
	if isFinish {
		prompt = "task finish"
		if fn.FinishCallback == nil {
			log.Consolef("%s not impl output impl\n", context.Task.Type)
			return
		}
		callback = fn.FinishCallback
	} else {
		prompt = "task done"
		if fn.DoneCallback == nil {
			log.Debugf("%s not impl output impl\n", context.Task.Type)
			return
		}
		callback = fn.DoneCallback
	}

	s := logs.GreenBold(fmt.Sprintf("[%s.%d] %s (%d/%d),%s\n",
		context.Task.SessionId, context.Task.TaskId, prompt,
		context.Task.Cur, context.Task.Total,
		message))
	log.Importantf(s)
	if callee != consts.CalleeCMD {
		return
	}
	var err error
	var resp string
	if isFinish {
		log.FileLog(s)
		resp, err = callback(context)
		log.FileLog(resp + "\n")
	} else {
		resp, err = callback(context)
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
			Log.Infof("%s has joined the game\n", event.Client.Name)
		} else if event.Op == consts.CtrlClientLeft {
			Log.Infof("%s left the game\n", event.Client.Name)
		}
	case consts.EventBroadcast:
		Log.Infof("%s : %s  %s\n", event.Client.Name, event.Message, event.Err)
	case consts.EventSession:
		s.handlerSession(event)
	case consts.EventNotify:
		Log.Importantf("%s notified: %s %s\n", event.Client.Name, event.Message, event.Err)
	case consts.EventJob:
		s.handleJob(event)
	case consts.EventListener:
		Log.Importantf("[%s] %s: %s %s\n", event.Type, event.Op, event.Message, event.Err)
	case consts.EventTask:
		s.handlerTask(event)
	case consts.EventWebsite:
		Log.Importantf("[%s] %s: %s %s\n", event.Type, event.Op, event.Message, event.Err)
	case consts.EventBuild:
		Log.Importantf("[%s] %s\n", event.Type, event.Message)
	case consts.EventPivot:
		Log.Importantf("[%s] %s: %s\n", event.Type, event.Op, event.Message)
	case consts.EventBof:
		Log.Importantf("%s\n%s\n", "Get Bof Message", event.Message)
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
	}
	switch pipeline.Body.(type) {
	case *clientpb.Pipeline_Tcp:
		Log.Importantf("[%s] %s: tcp %s on %s %s:%d\n", event.Type, event.Op,
			pipeline.Name, pipeline.ListenerId, pipeline.Ip, pipeline.GetTcp().Port)
	case *clientpb.Pipeline_Bind:
		Log.Importantf("[%s] %s: bind %s on %s %s\n", event.Type, event.Op,
			pipeline.Name, pipeline.ListenerId, pipeline.Ip)
	case *clientpb.Pipeline_Rem:
		Log.Importantf("[%s] %s: rem %s on %s %s:%d\n", event.Type, event.Op,
			pipeline.Name, pipeline.ListenerId, pipeline.Ip, pipeline.GetRem().Port)

		//Log.Infof("[%s] %s: rem -c %s \n", event.Type, event.Op, pipeline.GetRem().Link)
	case *clientpb.Pipeline_Web:
		if event.Op == consts.CtrlWebContentAdd {
			var root = ""
			if pipeline.GetWeb().Root != "/" {
				root = pipeline.GetWeb().Root
			}
			for _, content := range pipeline.GetWeb().Contents {
				Log.Importantf("[%s] %s: web %s on %s %d, routePath is %s\n", event.Type, event.Op,
					pipeline.ListenerId, pipeline.Name, pipeline.GetWeb().Port,
					fmt.Sprintf("http://%s:%d%s%s", pipeline.Ip, pipeline.GetWeb().Port, root, content.Path))
			}
			return
		}
		Log.Importantf("[%s] %s: web %s on %s %d, routePath is %s\n", event.Type, event.Op,
			pipeline.ListenerId, pipeline.Name, pipeline.GetWeb().Port,
			fmt.Sprintf("http://%s:%d%s", pipeline.Ip, pipeline.GetWeb().Port, pipeline.GetWeb().Root))
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
		Log.Important(logs.GreenBold(fmt.Sprintf("[%s]: %s \n", consts.CtrlSessionRegister, event.Message)))
	case consts.CtrlSessionTask:
		logs.Log.Infof(logs.GreenBold(fmt.Sprintf("[%s.%d] run task %s: %s\n", sid, event.Task.TaskId, event.Task.Type, event.Message)))
	case consts.CtrlSessionError:
		log := s.ObserverLog(sid)
		log.Errorf(logs.GreenBold(fmt.Sprintf("[%s] task: %d error: %s\n", sid, event.Task.TaskId, event.Err)))
	case consts.CtrlSessionLog:
		log := s.ObserverLog(sid)
		log.Errorf("[%s] log: \n%s\n", sid, event.Message)
	case consts.CtrlSessionLeave:
		Log.Importantf(logs.RedBold(fmt.Sprintf("[%s] session stop: %s\n", sid, event.Message)))
	case consts.CtrlSessionReborn:
		Log.Important(logs.GreenBold(fmt.Sprintf("[%s]: %s\n", consts.CtrlSessionReborn, event.Message)))
	}
}
