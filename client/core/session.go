package core

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/metadata"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
)

func NewSession(sess *clientpb.Session, server *ServerStatus) *Session {
	var data *types.SessionContext
	err := json.Unmarshal([]byte(sess.Data), &data)
	if err != nil {
		Log.Warnf("Failed to unmarshal session data: %v", err)
	}
	return &Session{
		ctx:      context.Background(),
		ctxValue: make(map[string]string),
		Session:  sess,
		Server:   server,
		Data:     data,
		Callee:   consts.CalleeCMD,
		Log:      NewLogger(filepath.Join(assets.GetLogDir(), fmt.Sprintf("%s.log", sess.SessionId))),
		Locker:   &sync.Mutex{},
	}
}

type Session struct {
	*clientpb.Session
	ctx      context.Context
	ctxValue map[string]string
	Data     *types.SessionContext
	Server   *ServerStatus
	Callee   string // cmd/mal/sdk
	LastTask *clientpb.Task
	Log      *Logger
	Locker   *sync.Mutex
}

func (s *Session) Clone(callee string) *Session {
	return &Session{
		Data:     s.Data,
		Session:  s.Session,
		Server:   s.Server,
		Callee:   callee,
		ctx:      s.ctx,
		ctxValue: s.ctxValue,
		Locker:   &sync.Mutex{},
	}
}

func (s *Session) Value(key string) (string, error) {
	if value, ok := s.ctxValue[key]; ok {
		return value, nil
	}
	return "", fmt.Errorf("key not found")
}

func (s *Session) WithValue(kv ...string) (*Session, error) {
	ctxValue := maps.Clone(s.ctxValue)
	if len(kv)%2 == 1 {
		return nil, fmt.Errorf("got the odd number of input pairs for metadata: %d", len(kv))
	}
	for i := 0; i < len(kv); i += 2 {
		key := strings.ToLower(kv[i])
		ctxValue[key] = kv[i+1]
	}

	return &Session{
		Data:     s.Data,
		Session:  s.Session,
		Server:   s.Server,
		Callee:   s.Callee,
		ctx:      s.ctx,
		ctxValue: ctxValue,
		Locker:   &sync.Mutex{},
	}, nil
}

func (s *Session) Context() context.Context {
	ss := []string{
		"session_id", s.SessionId,
		"callee", s.Callee,
	}
	for k, v := range s.ctxValue {
		ss = append(ss, k)
		ss = append(ss, v)
		delete(s.ctxValue, k)
	}
	return metadata.NewOutgoingContext(s.ctx, metadata.Pairs(ss...))
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
	profile, err := assets.GetProfile()
	if err != nil {
		Log.Errorf("Failed to get profile: %v", err)
		return
	}
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
