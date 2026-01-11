package core

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
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
	*client.ServerState
}

// NewServer wraps client.ServerState into core.ServerState
func NewServer(conn *grpc.ClientConn, config *mtls.ClientConfig) (*Server, error) {
	s, err := client.NewServerStatus(conn, config)
	if err != nil {
		return nil, err
	}
	ser := &Server{ServerState: s}
	events, err := ser.GetEvent(context.Background(), &clientpb.Int{})
	if err != nil {
		return nil, err
	}
	for _, event := range events.GetEvents() {
		ser.HandlerEvent(event)
	}
	return ser, nil
}

func (s *Server) AddDoneCallback(task *clientpb.Task, callback client.TaskCallback) {
	s.DoneCallbacks.Store(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId), callback)
}

func (s *Server) AddCallback(task *clientpb.Task, callback client.TaskCallback) {
	s.FinishCallbacks.Store(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId), callback)
}

func (s *Server) triggerTaskDone(event *clientpb.Event) {
	task := event.GetTask()
	sess, err := s.GetOrUpdateSession(event.Task.SessionId)
	if err != nil {
		client.Log.Errorf("session not found: %s\n", event.Task.SessionId)
		return
	}

	log := s.ObserverLog(event.Task.SessionId)
	err = types.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()) + "\n")
		return
	}
	taskContext := wrapToTaskContext(event)
	HandlerTask(sess, log, taskContext, event.Message, event.Callee, false)

	if callback, ok := s.DoneCallbacks.Load(fmt.Sprintf("%s-%d", task.SessionId, task.TaskId)); ok {
		callback.(client.TaskCallback)(taskContext)
	}
}

func (s *Server) triggerTaskFinish(event *clientpb.Event) {
	task := event.GetTask()
	sess, err := s.GetOrUpdateSession(event.Task.SessionId)
	if err != nil {
		client.Log.Errorf("session not found: %s\n", event.Task.SessionId)
		return
	}

	log := s.ObserverLog(event.Task.SessionId)
	err = types.HandleMaleficError(event.Spite)
	if err != nil {
		log.Errorf(logs.RedBold(err.Error()) + "\n")
		return
	}
	taskContext := wrapToTaskContext(event)
	HandlerTask(sess, log, taskContext, event.Message, event.Callee, true)

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

	if resp != "" && (callee == consts.CalleeCMD || callee == consts.CalleeMal || callee == consts.CalleeRPC || callee == consts.CalleeMCP) {
		if log != client.MuteLog {
			tui.Down(1)
			log.Console(resp + "\n")
		}
	}
}

func (s *Server) AddEventHook(event client.EventCondition, callback client.OnEventFunc) {
	if _, ok := s.EventHook[event]; !ok {
		s.EventHook[event] = []client.OnEventFunc{}
	}
	s.EventHook[event] = append(s.EventHook[event], callback)
}

func (s *Server) EventHandler() {
	eventStream, err := s.Rpc.Events(context.Background(), &clientpb.Empty{})
	if err != nil {
		return
	}
	s.Update()
	if s.GetInteractive() != nil {
		s.UpdateSession(s.GetInteractive().SessionId)
	}
	s.EventStatus = true
	client.Log.Info("starting event loop\n")
	defer func() {
		client.Log.Warnf("event stream broken\n")
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
								client.Log.Errorf("error running event hook: %s", err)
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
			client.Log.Info(event.Formatted + "\n")
		} else if event.Op == consts.CtrlClientLeft {
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
		client.Log.Important(event.Formatted + "\n")
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
	if event.Err != "" {
		client.Log.Errorf("[%s] %s: %s\n", event.Type, event.Op, event.Err)
		return
	}
	pipeline := event.GetJob().GetPipeline()
	switch event.Op {
	case consts.CtrlPipelineSync:
		s.Pipelines[pipeline.Name] = pipeline
	case consts.CtrlPipelineStop:
		delete(s.Pipelines, pipeline.Name)
		client.Log.Important(event.Formatted + "\n")
	default:
		client.Log.Important(event.Formatted + "\n")
	}
}

func (s *Server) handlerTask(event *clientpb.Event) {
	switch event.Op {
	case consts.CtrlTaskCallback:
		s.triggerTaskDone(event)
	case consts.CtrlTaskFinish:
		s.triggerTaskFinish(event)
	case consts.CtrlTaskCancel:
		log := s.ObserverLog(event.Task.SessionId)
		log.Importantf("[%s.%d] task canceled\n", event.Task.SessionId, event.Task.TaskId)
	case consts.CtrlTaskError:
		log := s.ObserverLog(event.Task.SessionId)
		log.Errorf("[%s.%d] %s\n", event.Task.SessionId, event.Task.TaskId, event.Err)
	}
}

func (s *Server) handlerSession(event *clientpb.Event) {
	sid := event.Session.SessionId
	switch event.Op {
	case consts.CtrlSessionRegister:
		s.AddSession(event.Session)
		client.Log.Important(event.Formatted + "\n")
	case consts.CtrlSessionUpdate:
		s.AddSession(event.Session)
	case consts.CtrlSessionTask:
		log := s.ObserverLog(sid)
		log.Info(event.Formatted + "\n")
	case consts.CtrlSessionError:
		log := s.ObserverLog(sid)
		log.Error(event.Formatted + "\n")
	case consts.CtrlSessionLog:
		log := s.ObserverLog(sid)
		log.Error(event.Formatted + "\n")
	case consts.CtrlSessionDead:
		client.Log.Important(event.Formatted + "\n")
	case consts.CtrlSessionReborn:
		s.AddSession(event.Session)
		client.Log.Important(event.Formatted + "\n")
	}
}
