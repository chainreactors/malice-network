package core

import (
	"context"
	"errors"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"google.golang.org/grpc"
	"sync"
)

type TaskCallback func(resp *implantpb.Spite)

func InitServerStatus(conn *grpc.ClientConn, config *mtls.ClientConfig) (*ServerStatus, error) {
	var err error
	s := &ServerStatus{
		Rpc: &Rpc{
			MaliceRPCClient:   clientrpc.NewMaliceRPCClient(conn),
			ListenerRPCClient: listenerrpc.NewListenerRPCClient(conn),
		},
		ActiveTarget:    &ActiveTarget{},
		Sessions:        make(map[string]*Session),
		Observers:       map[string]*Observer{},
		finishCallbacks: &sync.Map{},
		doneCallbacks:   &sync.Map{},
		EventHook:       make(map[intermediate.EventCondition][]intermediate.OnEventFunc),
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

	err = s.UpdateSessions(false)
	if err != nil {
		return nil, err
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
	Listeners       []*clientpb.Listener
	Sessions        map[string]*Session
	Observers       map[string]*Observer
	sessions        []*clientpb.Session
	finishCallbacks *sync.Map
	doneCallbacks   *sync.Map
	EventHook       map[intermediate.EventCondition][]intermediate.OnEventFunc
}

func (s *ServerStatus) AddSession(sess *clientpb.Session) {
	if origin, ok := s.Sessions[sess.SessionId]; ok {
		origin.Session = sess
	} else {
		s.Sessions[sess.SessionId] = NewSession(sess, s)
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
	s.Listeners = listeners.GetListeners()
	return nil
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
