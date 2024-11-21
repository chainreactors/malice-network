package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/tui"
	"io"
	"os"
)

func (s *ServerStatus) AddDoneCallback(task *clientpb.Task, callback TaskCallback) {
	s.doneCallbacks.Store(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId), callback)
}

func (s *ServerStatus) AddCallback(task *clientpb.Task, callback TaskCallback) {
	s.finishCallbacks.Store(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId), callback)
}

func (s *ServerStatus) triggerTaskDone(event *clientpb.Event) {
	task := event.GetTask()
	log := s.ObserverLog(event.Task.SessionId)
	err := handler.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()))
		return
	}
	if fn, ok := intermediate.InternalFunctions[event.Task.Type]; ok && fn.DoneCallback != nil {
		resp, err := fn.DoneCallback(&clientpb.TaskContext{
			Task:    event.Task,
			Session: event.Session,
			Spite:   event.Spite,
		})
		if err != nil {
			log.Errorf(logs.RedBold(err.Error()))
		} else {
			log.Importantf(logs.GreenBold(fmt.Sprintf("[%s.%d] task done (%d/%d): %s\n",
				event.Task.SessionId, event.Task.TaskId,
				event.Task.Cur, event.Task.Total, resp)))
		}
	} else {
		log.Debugf("%v\n", event.Spite)
	}

	if callback, ok := s.finishCallbacks.Load(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId)); ok {
		callback.(TaskCallback)(event.Spite)
	}
}

func (s *ServerStatus) triggerTaskFinish(event *clientpb.Event) {
	task := event.GetTask()
	log := s.ObserverLog(event.Task.SessionId)
	err := handler.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()))
		return
	}

	if fn, ok := intermediate.InternalFunctions[event.Task.Type]; ok && fn.FinishCallback != nil {
		log.Importantf(logs.GreenBold(fmt.Sprintf("[%s.%d] task finish (%d/%d),%s\n",
			event.Task.SessionId, event.Task.TaskId,
			event.Task.Cur, event.Task.Total,
			event.Message)))
		if event.Callee != consts.CalleeCMD {
			return
		}
		resp, err := fn.FinishCallback(&clientpb.TaskContext{
			Task:    event.Task,
			Session: event.Session,
			Spite:   event.Spite,
		})
		if err != nil {
			log.Errorf(logs.RedBold(err.Error()))
		} else {
			log.Console(resp + "\n")
		}
	} else {
		log.Consolef("%s not impl output impl\n", event.Task.Type)
	}

	callbackId := fmt.Sprintf("%s_%d", task.SessionId, task.TaskId)
	if callback, ok := s.finishCallbacks.Load(callbackId); ok {
		callback.(TaskCallback)(event.Spite)
		s.finishCallbacks.Delete(callbackId)
		s.doneCallbacks.Delete(callbackId)
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
	s.EventStatus = true
	Log.Importantf("starting event loop\n")
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
		// Trigger event based on type
		switch event.Type {
		case consts.EventClient:
			if event.Op == consts.CtrlClientJoin {
				Log.Infof("%s has joined the game", event.Client.Name)
			} else if event.Op == consts.CtrlClientLeft {
				Log.Infof("%s left the game", event.Client.Name)
			}
		case consts.EventBroadcast:
			Log.Infof("%s : %s  %s", event.Client.Name, event.Message, event.Err)
		case consts.EventSession:
			s.handlerSession(event)
		case consts.EventNotify:
			Log.Importantf("%s notified: %s %s", event.Client.Name, event.Message, event.Err)
		case consts.EventJob:
			if event.Err != "" {
				Log.Errorf("[%s] %s: %s", event.Type, event.Op, event.Err)
				continue
			}
			tui.Down(1)
			pipeline := event.GetJob().GetPipeline()
			switch pipeline.Body.(type) {
			case *clientpb.Pipeline_Tcp:
				Log.Importantf("[%s] %s: tcp %s on %s %s:%d", event.Type, event.Op,
					pipeline.Name, pipeline.ListenerId, pipeline.GetTcp().Host, pipeline.GetTcp().Port)
			case *clientpb.Pipeline_Web:
				Log.Importantf("[%s] %s: web %s on %s %d, routePath is %s", event.Type, event.Op,
					pipeline.ListenerId, pipeline.Name, pipeline.GetWeb().Port,
					pipeline.GetWeb().Root)
			}
		case consts.EventListener:
			tui.Down(1)
			Log.Importantf("[%s] %s: %s %s", event.Type, event.Op, event.Message, event.Err)
		case consts.EventTask:
			s.handlerTask(event)
		case consts.EventWebsite:
			tui.Down(1)
			Log.Importantf("[%s] %s: %s %s", event.Type, event.Op, event.Message, event.Err)
		case consts.EventBuild:
			tui.Down(1)
			Log.Importantf("[%s] %s", event.Type, event.Message)
		}
	}
}

func (s *ServerStatus) handlerTask(event *clientpb.Event) {
	tui.Down(1)
	switch event.Op {
	case consts.CtrlTaskCallback:
		s.triggerTaskDone(event)
	case consts.CtrlTaskFinish:
		s.triggerTaskFinish(event)
	case consts.CtrlTaskCancel:
		Log.Importantf("[%s.%d] task canceled", event.Task.SessionId, event.Task.TaskId)
	case consts.CtrlTaskError:
		Log.Errorf("[%s.%d] %s", event.Task.SessionId, event.Task.TaskId, event.Err)
	}
}

func (s *ServerStatus) handlerSession(event *clientpb.Event) {
	sid := event.Session.SessionId
	switch event.Op {
	case consts.CtrlSessionRegister:
		s.AddSession(event.Session)
		Log.Importantf("register session: %s \n", event.Message)
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
	}
}

func HandleTaskContext(log *Logger, context *clientpb.TaskContext, fn *intermediate.InternalFunc, writeToFile bool, logPath string) error {
	log.Importantf(logs.GreenBold(fmt.Sprintf("[%s.%d] task finish (%d/%d), %s\n",
		context.Task.SessionId, context.Task.TaskId,
		context.Task.Cur, context.Task.Total,
		context.Task.Description)))

	resp, err := fn.FinishCallback(context)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()))
		return err
	} else {
		log.Console(resp + "\n")
		if writeToFile {
			err := os.WriteFile(logPath, []byte(resp), os.ModePerm)
			if err != nil {
				log.Errorf("Error writing to file: %s", err)
				return err
			}
		}
	}
	return nil
}
