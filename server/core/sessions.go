package core

import (
	"errors"
	"sync"
	"time"
)

var (
	// Sessions - Manages implant connections
	Sessions = &sessions{
		sessions: &sync.Map{},
	}

	// ErrUnknownMessageType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessageType = errors.New("unknown message type")

	// ErrImplantTimeout - The implant did not respond prior to timeout deadline
	ErrImplantTimeout = errors.New("implant timeout")
)

// Session - Represents a connection to an implant
type Session struct {
	ID       string
	Name     string
	Hostname string
	Username string
	UUID     string
	UID      string
	GID      string
	OS       string
	Version  string
	Arch     string
	PID      int32
	Filename string
	//Connection        *ImplantConnection
	ActiveC2          string
	ReconnectInterval int64
	ProxyURL          string
	PollTimeout       int64
	Burned            bool
	Extensions        []string
	ConfigID          string
	PeerID            int64
	Locale            string
}

// Request
func (s *Session) Request(msgType uint32, timeout time.Duration, data []byte) ([]byte, error) {
	//resp := make(chan *implantpb.Spite)
	return nil, nil
}

// sessions - Manages the slivers, provides atomic access
type sessions struct {
	sessions *sync.Map // map[uint32]*Session
}

// All - Return a list of all sessions
func (s *sessions) All() []*Session {
	all := []*Session{}
	s.sessions.Range(func(key, value interface{}) bool {
		all = append(all, value.(*Session))
		return true
	})
	return all
}

// Get - Get a session by ID
func (s *sessions) Get(sessionID string) *Session {
	if val, ok := s.sessions.Load(sessionID); ok {
		return val.(*Session)
	}
	return nil
}

// Add - Add a sliver to the hive (atomically)
func (s *sessions) Add(session *Session) *Session {
	s.sessions.Store(session.ID, session)
	//EventBroker.Publish(Event{
	//	EventType: consts.SessionOpenedEvent,
	//	Session:   session,
	//})
	return session
}

// Remove - Remove a sliver from the hive (atomically)
func (s *sessions) Remove(sessionID string) {
	//val, ok := s.sessions.Load(sessionID)
	//if !ok {
	//	return
	//}
	//parentSession := val.(*Session)
	//children := findAllChildrenByPeerID(parentSession.PeerID)
	//s.sessions.Delete(parentSession.ID)
	//coreLog.Debugf("Removing %d children of session %d (%v)", len(children), parentSession.ID, children)
	//for _, child := range children {
	//	childSession, ok := s.sessions.LoadAndDelete(child.SessionID)
	//	if ok {
	//		PivotSessions.Delete(childSession.(*Session).Connection.ID)
	//		EventBroker.Publish(Event{
	//			EventType: consts.SessionClosedEvent,
	//			Session:   childSession.(*Session),
	//		})
	//	}
	//}

	// Remove the parent session
	//EventBroker.Publish(Event{
	//	EventType: consts.SessionClosedEvent,
	//	Session:   parentSession,
	//})
}

// NewSession - Create a new session
//func NewSession(implantConn *ImplantConnection) *Session {
//	implantConn.UpdateLastMessage()
//	return &Session{
//		ID:         nextSessionID(),
//		Connection: implantConn,
//	}
//}
