package core

import (
	"context"
	"fmt"
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
	var log *logs.Logger
	log = logs.NewLogger(LogLevel)
	log.SetFormatter(DefaultLogStyle)
	logFile, err := os.OpenFile(filepath.Join(assets.GetLogDir(), fmt.Sprintf("%s.log", sess.SessionId)), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		Log.Warnf("Failed to open log file: %v", err)
	}
	return &Session{
		Session: sess,
		Server:  server,
		Callee:  consts.CalleeCMD,
		Log:     &Logger{Logger: log, logFile: logFile},
	}
}

type Session struct {
	*clientpb.Session
	Server   *ServerStatus
	Callee   string // cmd/mal/sdk
	LastTask *clientpb.Task
	Log      *Logger
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
	s.LastTask = task
	_, err := s.Server.Rpc.SessionEvent(s.Context(), &clientpb.Event{
		Type:    consts.EventSession,
		Op:      consts.CtrlSessionTask,
		Task:    task,
		Session: s.Session,
		Client:  s.Server.Client,
		Message: []byte(msg),
	})
	if err != nil {
		Log.Errorf(err.Error() + "\n")
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
		Log.Errorf(err.Error() + "\n")
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
	for _, a := range s.Addons {
		if a.Name == addon {
			return s.HasDepend(a.Depend)
		}
	}
	return false
}

func (s *Session) GetAddon(name string) *implantpb.Addon {
	for _, a := range s.Addons {
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

func (s *Session) GetHistory() {
	profile := assets.GetProfile()
	contexts, err := s.Server.Rpc.GetSessionHistory(s.Context(), &clientpb.Int{
		Limit: int32(profile.Settings.MaxServerLogSize),
	})
	if err != nil {
		Log.Errorf("Failed to get session log: %v", err)
		return
	}
	logPath := assets.GetLogDir()
	logPath = filepath.Join(logPath, fmt.Sprintf("%s.log", s.SessionId))

	for _, context := range contexts.Contexts {
		HandlerTask(s, context, []byte{}, consts.CalleeCMD, true)
	}
}

type ActiveTarget struct {
	Session *Session
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
	return
}

// Background - Background the active session
func (s *ActiveTarget) Background() {
	s.Session = nil
}
