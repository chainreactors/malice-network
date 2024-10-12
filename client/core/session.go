package core

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"google.golang.org/grpc/metadata"
	"os"
	"path/filepath"
	"slices"
)

func NewSession(sess *clientpb.Session, server *ServerStatus) *Session {
	return &Session{
		Session: sess,
		Server:  server,
		Callee:  consts.CalleeCMD,
	}
}

type Session struct {
	*clientpb.Session
	Server *ServerStatus
	Callee string // cmd/mal/sdk
}

func (s *Session) Clone(callee string) *Session {
	return &Session{
		Session: s.Session,
		Server:  s.Server,
		Callee:  callee,
	}
}

func (s *Session) Context() context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", s.SessionId,
		"callee", s.Callee,
	),
	)
}

func (s *Session) Console(task *clientpb.Task, msg string) {
	_, err := s.Server.Rpc.SessionEvent(s.Context(), &clientpb.Event{
		Type:    consts.EventSession,
		Op:      consts.CtrlSessionTask,
		Task:    task,
		Session: s.Session,
		Client:  s.Server.Client,
		Message: []byte(msg),
	})
	if err != nil {
		Log.Errorf(err.Error())
	}
}

func (s *Session) Error(task *clientpb.Task, err error) {
	_, err = s.Server.Rpc.SessionEvent(s.Context(), &clientpb.Event{
		Type:    consts.EventSession,
		Op:      consts.CtrlSessionError,
		Task:    task,
		Session: s.Session,
		Client:  s.Server.Client,
		Err:     err.Error(),
	})
	if err != nil {
		Log.Errorf(err.Error())
	}
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

func (s *Session) HasAddon(addon string) bool {
	for _, a := range s.Addons.Addons {
		if a.Name == addon {
			return s.HasDepend(a.Depend)
		}
	}
	return false
}

func (s *Session) GetAddon(name string) *implantpb.Addon {
	for _, a := range s.Addons.Addons {
		if a.Name == name {
			return a
		}
	}
	return nil
}

func (s *Session) HasTask(taskId uint32) bool {
	for _, task := range s.Tasks.Tasks {
		if task.TaskId == taskId {
			return true
		}
	}
	return false
}

func (s *Session) GetLog() {
	log, err := s.Server.Rpc.SyncLog(s.Context(), &clientpb.Sync{
		FileId: s.SessionId,
	})
	if err != nil {
		Log.Errorf("Can't get log file: %s", err)
		return
	}
	logPath := assets.GetProfile().TempDir
	if logPath == "" {
		logPath = assets.GetTempDir()
	}
	logPath = filepath.Join(logPath, log.Name)
	err = os.WriteFile(logPath, log.Content, 0644)

	if err != nil {
		Log.Errorf("Can't write log file: %s", err)
		return
	}
}

func NewObserver(session *Session) *Observer {
	return &Observer{
		Session: session,
		Log:     Log,
	}
}

// Observer - A function to call when the sessions changes
type Observer struct {
	*Session
	Log *Logger
}

func (o *Observer) SessionId() string {
	return o.GetSessionId()
}

type ActiveTarget struct {
	Session  *Session
	Observer *Observer
}

func (s *ActiveTarget) GetInteractive() *Session {
	if s.Session == nil {
		logs.Log.Warn("Please select a session or beacon via `use`")
		return nil
	}
	return s.Session
}

// GetSessionInteractive - Get the active target(s)
func (s *ActiveTarget) Get() *Session {
	return s.Session
}

func (s *ActiveTarget) Context() context.Context {
	if s.Session != nil {
		return s.Session.Context()
	} else {
		return nil
	}
}

// Set - Change the active session
func (s *ActiveTarget) Set(session *Session) {
	s.Session = session
	s.Observer = NewObserver(session)
	return
}

// Background - Background the active session
func (s *ActiveTarget) Background() {
	s.Session = nil
	s.Observer = nil
}
