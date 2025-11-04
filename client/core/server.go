package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/types"
	"io"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/session"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"google.golang.org/grpc"
)

var ErrLuaVMDead = fmt.Errorf("lua vm is dead")

func wrapToTaskContext(event *clientpb.Event) *clientpb.TaskContext {
	return &clientpb.TaskContext{
		Task:    event.Task,
		Session: event.Session,
		Spite:   event.Spite,
	}
}

type Server struct {
	*session.ServerStatus
}

// NewServer wraps session.ServerStatus into core.ServerStatus
func NewServer(conn *grpc.ClientConn, config *mtls.ClientConfig) (*Server, error) {
	s, err := session.InitServerStatus(conn, config)
	if err != nil {
		return nil, err
	}
	events, err := s.GetEvent(context.Background(), &clientpb.Int{})
	if err != nil {
		return nil, err
	}
	for _, event := range events.GetEvents() {
		s.HandlerEvent(event)
	}
	return &Server{ServerStatus: s}, nil
}

func (s *Server) AddDoneCallback(task *clientpb.Task, callback session.TaskCallback) {
	s.DoneCallbacks.Store(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId), callback)
}

func (s *Server) AddCallback(task *clientpb.Task, callback session.TaskCallback) {
	s.FinishCallbacks.Store(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId), callback)
}

func (s *Server) triggerTaskDone(event *clientpb.Event) {
	task := event.GetTask()
	sess, err := s.GetOrUpdateSession(event.Task.SessionId)
	if err != nil {
		session.Log.Errorf("session not found: %s\n", event.Task.SessionId)
		return
	}

	log := s.ObserverLog(event.Task.SessionId)
	err = types.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()) + "\n")
		return
	}
	taskContext := wrapToTaskContext(event)
	HandlerTask(sess, taskContext, event.Message, event.Callee, false)

	if callback, ok := s.DoneCallbacks.Load(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId)); ok {
		callback.(session.TaskCallback)(taskContext)
	}
}

func (s *Server) triggerTaskFinish(event *clientpb.Event) {
	task := event.GetTask()
	sess, err := s.GetOrUpdateSession(event.Task.SessionId)
	if err != nil {
		session.Log.Errorf("session not found: %s\n", event.Task.SessionId)
		return
	}

	log := s.ObserverLog(event.Task.SessionId)
	err = types.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()) + "\n")
		return
	}
	taskContext := wrapToTaskContext(event)
	HandlerTask(sess, taskContext, event.Message, event.Callee, true)

	callbackId := fmt.Sprintf("%s-%d", task.SessionId, task.TaskId)
	if callback, ok := s.FinishCallbacks.Load(callbackId); ok {
		callback.(session.TaskCallback)(taskContext)
		s.FinishCallbacks.Delete(callbackId)
		s.DoneCallbacks.Delete(callbackId)
	}
}

func HandlerTask(sess *session.Session, ctx *clientpb.TaskContext, message []byte, callee string, isFinish bool) {
	sess.Locker.Lock()
	defer sess.Locker.Unlock()
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
		log.Importantf("task %d %t\n", ctx.Task.TaskId, status)
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

	if callee != consts.CalleePty {
		log.Importantf(s)
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
		return
	}

	if resp != "" && callee == consts.CalleeCMD {
		tui.Down(1)
		log.Console(resp + "\n")
	}
}

func (s *Server) AddEventHook(event session.EventCondition, callback session.OnEventFunc) {
	if _, ok := s.EventHook[event]; !ok {
		s.EventHook[event] = []session.OnEventFunc{}
	}
	s.EventHook[event] = append(s.EventHook[event], callback)
}

func (s *Server) EventHandler() {
	eventStream, err := s.Rpc.Events(context.Background(), &clientpb.Empty{})
	if err != nil {
		return
	}
	s.Update()
	s.EventStatus = true
	session.Log.Info("starting event loop\n")
	defer func() {
		session.Log.Warnf("event stream broken\n")
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
					for i, fn := range fns {
						_, err := fn(event)
						if err != nil {
							if errors.Is(err, ErrLuaVMDead) {
								s.EventHook[condition] = append(fns[:i], fns[i+1:]...)
							} else {
								session.Log.Errorf("error running event hook: %s", err)
							}
						}
					}
				}()
			}
		}

		if fn, ok := s.EventCallback[event.Op]; ok {
			fn(event)
		}
		go func() {
			s.HandlerEvent(event)
		}()
	}
}

func (s *Server) HandlerEvent(event *clientpb.Event) {
	switch event.Type {
	case consts.EventClient:
		if event.Op == consts.CtrlClientJoin {
			session.Log.Info(event.Formatted + "\n")
		} else if event.Op == consts.CtrlClientLeft {
			session.Log.Info(event.Formatted + "\n")
		}
	case consts.EventBroadcast:
		session.Log.Info(event.Formatted + "\n")
	case consts.EventSession:
		s.handlerSession(event)
	case consts.EventNotify:
		session.Log.Important(event.Formatted + "\n")
	case consts.EventJob:
		s.handleJob(event)
	case consts.EventListener:
		session.Log.Important(event.Formatted + "\n")
	case consts.EventTask:
		s.handlerTask(event)
	case consts.EventWebsite:
		session.Log.Important(event.Formatted + "\n")
	case consts.EventBuild:
		session.Log.Important(event.Formatted + "\n")
	case consts.EventPivot:
		session.Log.Important(event.Formatted + "\n")
	case consts.EventContext:
		session.Log.Important(event.Formatted + "\n")
	case consts.EventCert:
		session.Log.Important(event.Formatted + "\n")
	}
}

func (s *Server) handleJob(event *clientpb.Event) {
	if event.Err != "" {
		session.Log.Errorf("[%s] %s: %s\n", event.Type, event.Op, event.Err)
		return
	}
	pipeline := event.GetJob().GetPipeline()
	switch event.Op {
	case consts.CtrlPipelineSync:
		s.Pipelines[pipeline.Name] = pipeline
	case consts.CtrlPipelineStop:
		delete(s.Pipelines, pipeline.Name)
		session.Log.Important(event.Formatted + "\n")
	default:
		session.Log.Important(event.Formatted + "\n")
	}
}

func (s *Server) handlerTask(event *clientpb.Event) {
	switch event.Op {
	case consts.CtrlTaskCallback:
		s.triggerTaskDone(event)
	case consts.CtrlTaskFinish:
		s.triggerTaskFinish(event)
	case consts.CtrlTaskCancel:
		session.Log.Importantf("[%s.%d] task canceled\n", event.Task.SessionId, event.Task.TaskId)
	case consts.CtrlTaskError:
		session.Log.Errorf("[%s.%d] %s\n", event.Task.SessionId, event.Task.TaskId, event.Err)
	}
}

func (s *Server) handlerSession(event *clientpb.Event) {
	sid := event.Session.SessionId
	switch event.Op {
	case consts.CtrlSessionRegister:
		s.AddSession(event.Session)
		session.Log.Important(event.Formatted + "\n")
	case consts.CtrlSessionUpdate:
		s.AddSession(event.Session)
	case consts.CtrlSessionTask:
		session.Log.Info(event.Formatted + "\n")
	case consts.CtrlSessionError:
		log := s.ObserverLog(sid)
		log.Error(event.Formatted + "\n")
	case consts.CtrlSessionLog:
		log := s.ObserverLog(sid)
		log.Error(event.Formatted + "\n")
	case consts.CtrlSessionDead:
		session.Log.Important(event.Formatted + "\n")
	case consts.CtrlSessionReborn:
		s.AddSession(event.Session)
		session.Log.Important(event.Formatted + "\n")
	}
}
