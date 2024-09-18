package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/handler"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/tui"
	"google.golang.org/grpc"
	"io"
	"sync"
)

type TaskCallback func(resp *implantpb.Spite)

func InitServerStatus(conn *grpc.ClientConn, config *mtls.ClientConfig) (*ServerStatus, error) {
	var err error
	s := &ServerStatus{
		Rpc:             clientrpc.NewMaliceRPCClient(conn),
		LisRpc:          listenerrpc.NewListenerRPCClient(conn),
		ActiveTarget:    &ActiveTarget{},
		Sessions:        make(map[string]*Session),
		Observers:       map[string]*Observer{},
		finishCallbacks: &sync.Map{},
		doneCallbacks:   &sync.Map{},
	}
	client, err := s.Rpc.LoginClient(context.Background(), &clientpb.LoginReq{
		Name: config.Operator,
		Host: config.LHost,
		Port: uint32(config.LPort),
	})
	if err != nil {
		return nil, err
	}
	s.Client = client
	s.Info, err = s.Rpc.GetBasic(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}

	clients, err := s.Rpc.GetClients(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, client := range clients.GetClients() {
		s.Clients = append(s.Clients, client)
	}

	listeners, err := s.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, listener := range listeners.GetListeners() {
		s.Listeners = append(s.Listeners, listener)
	}

	err = s.UpdateSessions(true)
	if err != nil {
		return nil, err
	}

	go s.EventHandler()

	return s, nil
}

type ServerStatus struct {
	Rpc    clientrpc.MaliceRPCClient
	LisRpc listenerrpc.ListenerRPCClient
	Info   *clientpb.Basic
	Client *clientpb.Client
	*ActiveTarget
	Clients         []*clientpb.Client
	Listeners       []*clientpb.Listener
	Sessions        map[string]*Session
	Observers       map[string]*Observer
	sessions        []*clientpb.Session
	finishCallbacks *sync.Map
	doneCallbacks   *sync.Map
}

func (s *ServerStatus) UpdateSessions(all bool) error {
	var sessions *clientpb.Sessions
	var err error
	if s == nil {
		return errors.New("You need login first")
	}
	if all {
		sessions, err = s.Rpc.GetSessions(context.Background(), &clientpb.Empty{})
	} else {
		sessions, err = s.Rpc.GetAlivedSessions(context.Background(), &clientpb.Empty{})
	}
	if err != nil {
		return err
	}
	s.sessions = sessions.Sessions
	newSessions := make(map[string]*Session)

	for _, session := range sessions.GetSessions() {
		if rawSess, ok := s.Sessions[session.SessionId]; ok {
			rawSess.Session = session
			newSessions[session.SessionId] = rawSess
		} else {
			newSessions[session.SessionId] = NewSession(session, s)
		}
	}

	s.Sessions = newSessions
	return nil
}

func (s *ServerStatus) UpdateSession(sid string) (*clientpb.Session, error) {
	session, err := s.Rpc.GetSession(context.Background(), &clientpb.SessionRequest{SessionId: sid})
	if err != nil {
		return nil, err
	}
	if rawSess, ok := s.Sessions[session.SessionId]; ok {
		rawSess.Session = session
	} else {
		s.Sessions[session.SessionId] = NewSession(session, s)
	}

	return nil, nil
}

func (s *ServerStatus) AlivedSessions() []*clientpb.Session {
	var alivedSessions []*clientpb.Session
	for _, session := range s.sessions {
		if session.IsAlive {
			alivedSessions = append(alivedSessions, session)
		}
	}
	return alivedSessions
}

func (s *ServerStatus) UpdateTasks(session *Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	tasks, err := s.Rpc.GetTasks(context.Background(), session.Session)
	if err != nil {
		return err
	}

	session.Tasks = &clientpb.Tasks{Tasks: tasks.Tasks}
	return nil
}

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
			log.Importantf(logs.GreenBold(fmt.Sprintf("[%s.%d] task done (%d/%d): %s",
				event.Task.SessionId, event.Task.TaskId,
				event.Task.Cur, event.Task.Total, resp)))
		}
	} else {
		log.Consolef("%v\n", event.Spite)
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
		log.Importantf(logs.GreenBold(fmt.Sprintf("[%s.%d] task finish (%d/%d), %s",
			event.Task.SessionId, event.Task.TaskId,
			event.Task.Cur, event.Task.Total,
			event.Message)))
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
		log.Consolef("%v\n", event.Spite.GetBody())
	}

	callbackId := fmt.Sprintf("%s_%d", task.SessionId, task.TaskId)
	if callback, ok := s.finishCallbacks.Load(callbackId); ok {
		callback.(TaskCallback)(event.Spite)
		s.finishCallbacks.Delete(callbackId)
		s.doneCallbacks.Delete(callbackId)
	}
}

func (s *ServerStatus) EventHandler() {
	defer Log.Warnf("event stream broken")
	eventStream, err := s.Rpc.Events(context.Background(), &clientpb.Empty{})
	if err != nil {
		logs.Log.Warnf("Error getting event stream: %v", err)
		return
	}
	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			continue
		}

		// Trigger event based on type
		switch event.Type {
		case consts.EventJoin:
			tui.Down(0)
			Log.Infof("%s has joined the game", event.Client.Name)
		case consts.EventLeft:
			tui.Down(0)
			Log.Infof("%s left the game", event.Client.Name)
		case consts.EventBroadcast:
			tui.Down(0)
			Log.Infof("%s : %s  %s", event.Client.Name, event.Message, event.Err)
		case consts.EventSession:
			tui.Down(0)
			s.handlerSession(event)
		case consts.EventNotify:
			tui.Down(0)
			Log.Importantf("%s notified: %s %s", event.Client.Name, event.Message, event.Err)
		case consts.EventJob:
			tui.Down(0)
			Log.Importantf("[%s] %s: %s %s", event.Type, event.Op, event.Message, event.Err)
		case consts.EventListener:
			tui.Down(0)
			Log.Importantf("[%s] %s: %s %s", event.Type, event.Op, event.Message, event.Err)
		case consts.EventTask:
			s.handlerTask(event)
		case consts.EventWebsite:
			tui.Down(0)
			Log.Importantf("[%s] %s: %s %s", event.Type, event.Op, event.Message, event.Err)
		}
		//con.triggerReactions(event)
	}
}

func (s *ServerStatus) handlerTask(event *clientpb.Event) {
	tui.Down(0)
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

func (s *ServerStatus) AddObserver(session *Session) string {
	Log.Infof("Add observer to %s", session.SessionId)
	s.Observers[session.SessionId] = &Observer{session, Log}
	return session.SessionId
}

func (s *ServerStatus) RemoveObserver(observerID string) {
	delete(s.Observers, observerID)
}

func (s *ServerStatus) ObserverLog(sessionId string) *Logger {
	if s.Session != nil && s.Session.SessionId == sessionId {
		return s.Observer.Log
	}

	if observer, ok := s.Observers[sessionId]; ok {
		return observer.Log
	}
	return MuteLog
}

func (s *ServerStatus) handlerSession(event *clientpb.Event) {
	switch event.Op {
	case consts.CtrlSessionRegister:
		Log.Importantf("%s session: %s ", event.Session.SessionId, event.Message)
	case consts.CtrlSessionConsole:
		log := s.ObserverLog(event.Task.SessionId)
		log.Importantf(logs.GreenBold(fmt.Sprintf("[%s.%d] run task %s: %s", event.Task.SessionId, event.Task.TaskId, event.Task.Type, event.Message)))
	case consts.CtrlSessionError:
		log := s.ObserverLog(event.Task.SessionId)
		log.Errorf(logs.GreenBold(fmt.Sprintf("[%s] task: %d error: %s\n", event.Task.SessionId, event.Task.TaskId, event.Err)))
	}
}
