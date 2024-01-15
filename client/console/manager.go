package console

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"google.golang.org/grpc/metadata"
)

// Observer - A function to call when the sessions changes
type Observer func(*clientpb.Session)

type ActiveTarget struct {
	session    *clientpb.Session
	observers  map[int]Observer
	observerID int
}

func (s *ActiveTarget) GetInteractive() *clientpb.Session {
	if s.session == nil {
		logs.Log.Warn("Please select a session or beacon via `use`")
		return nil
	}
	return s.session
}

// GetSessionInteractive - Get the active target(s)
func (s *ActiveTarget) Get() *clientpb.Session {
	return s.session
}

// GetSessionInteractive - GetSessionInteractive the active session
func (s *ActiveTarget) GetSessionInteractive() *clientpb.Session {
	if s.session == nil {
		logs.Log.Warn("Please select a session via `use`")
		return nil
	}
	return s.session
}

// GetSession - Same as GetSession() but doesn't print a warning
func (s *ActiveTarget) GetSession() *clientpb.Session {
	return s.session
}

// AddObserver - Observers to notify when the active session changes
func (s *ActiveTarget) AddObserver(observer Observer) int {
	s.observerID++
	s.observers[s.observerID] = observer
	return s.observerID
}

func (s *ActiveTarget) RemoveObserver(observerID int) {
	delete(s.observers, observerID)
}

func (s *ActiveTarget) Context() context.Context {
	if s.session != nil {
		return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
			"session_id", s.session.SessionId),
		)
	} else {
		return nil
	}
}

// Set - Change the active session
func (s *ActiveTarget) Set(session *clientpb.Session) {
	s.session = session
	for _, observer := range s.observers {
		observer(s.session)
	}
	return
}

// Background - Background the active session
func (s *ActiveTarget) Background() {
	s.session = nil
	for _, observer := range s.observers {
		observer(nil)
	}
}
