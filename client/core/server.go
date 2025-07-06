package core

import (
	"context"
	"errors"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"google.golang.org/grpc"
	"sync"
)

type TaskCallback func(resp *clientpb.TaskContext)

func InitServerStatus(conn *grpc.ClientConn, config *mtls.ClientConfig) (*ServerStatus, error) {
	var err error
	s := &ServerStatus{
		Rpc: &Rpc{
			MaliceRPCClient:   clientrpc.NewMaliceRPCClient(conn),
			ListenerRPCClient: listenerrpc.NewListenerRPCClient(conn),
		},
		ActiveTarget:    &ActiveTarget{},
		Listeners:       make(map[string]*clientpb.Listener),
		Pipelines:       make(map[string]*clientpb.Pipeline),
		Sessions:        make(map[string]*Session),
		Observers:       make(map[string]*Session),
		finishCallbacks: &sync.Map{},
		doneCallbacks:   &sync.Map{},
		EventHook:       make(map[intermediate.EventCondition][]intermediate.OnEventFunc),
		EventCallback:   make(map[string]func(*clientpb.Event)),
	}
	client, err := s.Rpc.LoginClient(context.Background(), &clientpb.LoginReq{
		Name: config.Operator,
		Host: config.Host,
		Port: uint32(config.Port),
	})
	if err != nil {
		return nil, err
	}
	s.Client = client
	s.Info, err = s.Rpc.GetBasic(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}

	err = s.Update()
	if err != nil {
		return nil, err
	}

	events, err := s.GetEvent(context.Background(), &clientpb.Int{})
	if err != nil {
		return nil, err
	}
	for _, event := range events.GetEvents() {
		s.handlerEvent(event)
	}
	return s, nil
}

type Rpc struct {
	clientrpc.MaliceRPCClient
	listenerrpc.ListenerRPCClient
}

type ServerStatus struct {
	*Rpc
	Info   *clientpb.Basic
	Client *clientpb.Client
	*ActiveTarget
	Clients         []*clientpb.Client
	Listeners       map[string]*clientpb.Listener
	Pipelines       map[string]*clientpb.Pipeline
	Sessions        map[string]*Session
	Observers       map[string]*Session
	sessions        []*clientpb.Session
	finishCallbacks *sync.Map
	doneCallbacks   *sync.Map
	EventStatus     bool
	EventHook       map[intermediate.EventCondition][]intermediate.OnEventFunc
	EventCallback   map[string]func(*clientpb.Event)
}

func (s *ServerStatus) Update() error {
	clients, err := s.Rpc.GetClients(context.Background(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	for _, client := range clients.GetClients() {
		s.Clients = append(s.Clients, client)
	}

	err = s.UpdateListener()
	if err != nil {
		return err
	}

	err = s.UpdatePipeline()
	if err != nil {
		return err
	}

	err = s.UpdateSessions(false)
	if err != nil {
		return err
	}
	return nil
}

func (s *ServerStatus) AddSession(sess *clientpb.Session) *Session {
	if origin, ok := s.Sessions[sess.SessionId]; ok {
		origin.Session = sess
		return origin
	} else {
		s.Sessions[sess.SessionId] = NewSession(sess, s)
		return s.Sessions[sess.SessionId]
	}
}

func (s *ServerStatus) UpdateSessions(all bool) error {
	var sessions *clientpb.Sessions
	var err error
	if s == nil {
		return errors.New("You need login first")
	}
	sessions, err = s.Rpc.GetSessions(context.Background(), &clientpb.SessionRequest{
		All: all,
	})
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

func (s *ServerStatus) UpdateSession(sid string) (*Session, error) {
	session, err := s.Rpc.GetSession(context.Background(), &clientpb.SessionRequest{SessionId: sid})
	if err != nil {
		return nil, err
	}
	if rawSess, ok := s.Sessions[session.SessionId]; ok {
		rawSess.Session = session
		return rawSess, nil
	} else {
		newSess := NewSession(session, s)
		s.Sessions[session.SessionId] = newSess
		return newSess, nil
	}
}

func (s *ServerStatus) GetLocalSession(sid string) (*Session, bool) {
	if sess, ok := s.Sessions[sid]; ok {
		return sess, true
	} else {
		return nil, false
	}
}

func (s *ServerStatus) GetOrUpdateSession(sid string) (*Session, error) {
	if sess, ok := s.Sessions[sid]; ok {
		return sess, nil
	} else {
		return s.UpdateSession(sid)
	}
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
	tasks, err := s.Rpc.GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: session.SessionId,
	})
	if err != nil {
		return err
	}

	session.Tasks = &clientpb.Tasks{Tasks: tasks.Tasks}
	return nil
}

func (s *ServerStatus) UpdateListener() error {
	listeners, err := s.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	for _, listener := range listeners.GetListeners() {
		s.Listeners[listener.Id] = listener
	}
	return nil
}

func (s *ServerStatus) UpdatePipeline() error {
	pipelines, err := s.Rpc.ListPipelines(context.Background(), &clientpb.Listener{})
	if err != nil {
		return err
	}
	for _, pipeline := range pipelines.GetPipelines() {
		s.Pipelines[pipeline.Name] = pipeline
	}
	return nil
}

func (s *ServerStatus) AddObserver(session *Session) string {
	Log.Infof("Add observer to %s", session.SessionId)
	s.Observers[session.SessionId] = session
	return session.SessionId
}

func (s *ServerStatus) RemoveObserver(observerID string) {
	delete(s.Observers, observerID)
}

func (s *ServerStatus) ObserverLog(sessionId string) *Logger {
	if s.Session != nil && s.Session.SessionId == sessionId {
		return s.Session.Log
	}

	if observer, ok := s.Observers[sessionId]; ok {
		return observer.Log
	}
	return MuteLog
}
