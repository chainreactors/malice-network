package repl

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"google.golang.org/grpc/metadata"
	"slices"
)

func NewSession(sess *clientpb.Session) *Session {
	return &Session{
		Session: sess,
		Log:     logs.NewLogger(LogLevel),
	}
}

type Session struct {
	*clientpb.Session
	Log *logs.Logger
}

func (s *Session) HasDepend(module string) bool {
	if alias, ok := consts.ModuleAliases[module]; ok {
		module = alias
	}
	if slices.Contains(s.Modules, module) {
		return true
	}
	return false
}

func (s *Session) Context() context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", s.SessionId),
	)
}

func NewObserver(session *Session) *Observer {
	return &Observer{
		Session: session,
	}
}

// Observer - A function to call when the sessions changes
type Observer struct {
	*Session
}

func (o *Observer) SessionId() string {
	return o.GetSessionId()
}

type ActiveTarget struct {
	session        *Session
	activeObserver *Observer
	callback       func(*Session)
}

func (s *ActiveTarget) GetInteractive() *Session {
	if s.session == nil {
		logs.Log.Warn("Please select a session or beacon via `use`")
		return nil
	}
	return s.session
}

// GetSessionInteractive - Get the active target(s)
func (s *ActiveTarget) Get() *Session {
	return s.session
}

func (s *ActiveTarget) Context() context.Context {
	if s.session != nil {
		return s.session.Context()
	} else {
		return nil
	}
}

// Set - Change the active session
func (s *ActiveTarget) Set(session *Session) {
	s.session = session
	s.callback(session)
	return
}

// Background - Background the active session
func (s *ActiveTarget) Background() {
	s.session = nil
	s.callback(nil)
}
