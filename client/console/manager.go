package console

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"google.golang.org/grpc/metadata"
)

func NewObserver(session *clientpb.Session) *Observer {
	return &Observer{
		session: session,
		log:     logs.NewLogger(LogLevel),
	}
}

// Observer - A function to call when the sessions changes
type Observer struct {
	session *clientpb.Session
	log     *logs.Logger
}

func (o *Observer) Logger() *logs.Logger {
	return o.log
}

func (o *Observer) SessionId() string {
	return o.session.SessionId
}

type ActiveTarget struct {
	session        *clientpb.Session
	activeObserver *Observer
	callback       func(*clientpb.Session)
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
	s.callback(session)
	s.session = session
	return
}

// Background - Background the active session
func (s *ActiveTarget) Background() {
	s.session = nil
	s.callback(nil)
}
