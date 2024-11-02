package core

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/gookit/config/v2"
	"google.golang.org/grpc"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// Sessions - Manages implant connections
	Sessions         = NewSessions()
	ExtensionModules = []string{consts.ModuleExecuteBof, consts.ModuleExecuteDll}
	// ErrUnknownMessageType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessageType = errors.New("unknown message type")

	// ErrImplantSendTimeout - The implant did not respond prior to timeout deadline
	ErrImplantSendTimeout = errors.New("implant timeout")
)

func NewSessions() *sessions {
	newSessions := &sessions{
		active: &sync.Map{},
	}
	ticker := NewTicker()
	_, err := ticker.Start(60, func() {
		for _, session := range newSessions.All() {
			currentTime := time.Now()
			timeDiff := currentTime.Unix() - int64(session.Timer.LastCheckin)
			isAlive := timeDiff <= int64(1+session.Timer.Interval)*3
			if !isAlive {
				newSessions.Remove(session.ID)
				EventBroker.Publish(Event{
					EventType: consts.EventSession,
					Op:        consts.CtrlSessionStop,
					Session:   session.ToProtobuf(),
					IsNotify:  true,
					Message:   fmt.Sprintf("session %s from %s at %s has stoped ", session.ID, session.Target, session.PipelineID),
				})
			}
		}
	})
	if err != nil {
		logs.Log.Errorf("cannot start ticker, %s", err.Error())
	}
	return newSessions
}

func NewSessionContext(req *clientpb.RegisterSession) *SessionContext {
	return &SessionContext{
		Modules: req.RegisterData.Module,
		Addons:  req.RegisterData.Addon.Addons,
		Loot:    map[string]string{},
		Argue:   map[string]string{},
		Data:    map[string]string{},
	}
}

type SessionContext struct {
	Modules []string
	Addons  []*implantpb.Addon
	Loot    map[string]string
	Argue   map[string]string
	Data    map[string]string
}

func (ctx *SessionContext) Update(req *clientpb.RegisterSession) {
	ctx.Modules = req.RegisterData.Module
	ctx.Addons = req.RegisterData.Addon.Addons
}

func NewSession(req *clientpb.RegisterSession) *Session {
	sess := &Session{
		Type:           req.Type,
		Name:           req.RegisterData.Name,
		Group:          "default",
		ProxyURL:       req.RegisterData.Proxy,
		ID:             req.SessionId,
		RawID:          req.RawId,
		PipelineID:     req.ListenerId,
		Target:         req.Target,
		IsPrivilege:    req.RegisterData.Sysinfo.IsPrivilege,
		Timer:          req.RegisterData.Timer,
		Tasks:          &Tasks{active: &sync.Map{}},
		Cache:          NewCache(10*consts.KB, path.Join(configs.CachePath, req.SessionId+".gob")),
		SessionContext: NewSessionContext(req),
		responses:      &sync.Map{},
	}
	logDir := filepath.Join(configs.LogPath, sess.ID)
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		logs.Log.Errorf("cannot create log directory %s, %s", logDir, err.Error())
	}
	if req.RegisterData.Sysinfo != nil {
		sess.UpdateSysInfo(req.RegisterData.Sysinfo)
	}

	return sess
}

// Session - Represents a connection to an implant
type Session struct {
	Type        string
	PipelineID  string
	ListenerID  string
	ID          string
	RawID       []byte
	Name        string
	Group       string
	Target      string
	IsPrivilege bool
	Os          *implantpb.Os
	Process     *implantpb.Process
	Timer       *implantpb.Timer
	Filepath    string
	WordDir     string
	ProxyURL    string
	Locale      string
	*SessionContext
	Tasks   *Tasks // task manager
	taskseq uint32
	*Cache
	responses *sync.Map
	rpcLog    *logs.Logger
}

func (s *Session) RpcLogger() *logs.Logger {
	var err error
	if s.rpcLog == nil {
		if auditLevel := config.Int(consts.ConfigAuditLevel); auditLevel > 0 {
			s.rpcLog, err = logs.NewFileLogger(filepath.Join(configs.AuditPath, s.ID+".log"))
			if err == nil {
				s.rpcLog.SuffixFunc = func() string {
					return time.Now().Format("2006-01-02 15:04.05")
				}
				if auditLevel == 2 {
					s.rpcLog.SetLevel(logs.Debug)
				}
			}
		}
	}
	return s.rpcLog
}

func (s *Session) TaskLog(task *Task, spite []byte) error {
	id := strconv.FormatUint(uint64(task.Id), 10)
	cur := strconv.FormatUint(uint64(task.Cur), 10)
	filePath := filepath.Join(configs.LogPath, s.ID, id+"_"+cur)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(string(spite))
	return err
}

func (s *Session) ToProtobuf() *clientpb.Session {
	currentTime := time.Now()
	timeDiff := currentTime.Unix() - int64(s.Timer.LastCheckin)
	isAlive := timeDiff <= int64(1+s.Timer.Interval)*3
	if s.Type != consts.BindPipeline {
		currentTime := time.Now()
		timeDiff := currentTime.Unix() - int64(s.LastCheckin)
		isAlive = timeDiff <= int64(s.Timer.Interval)*3
	} else {
		isAlive = true
	}

	return &clientpb.Session{
		Type:        s.Type,
		SessionId:   s.ID,
		RawId:       s.RawID,
		Note:        s.Name,
		GroupName:   s.Group,
		IsAlive:     isAlive,
		IsPrivilege: s.IsPrivilege,
		Target:      s.Target,
		PipelineId:  s.PipelineID,
		Os:          s.Os,
		Process:     s.Process,
		Timer:       s.Timer,
		Tasks:       s.Tasks.ToProtobuf(),
		Modules:     s.Modules,
		Addons:      s.Addons,
	}
}

func (s *Session) ToProtobufLite() *clientpb.Session {
	return &clientpb.Session{
		Type:        s.Type,
		SessionId:   s.ID,
		RawId:       s.RawID,
		Note:        s.Name,
		GroupName:   s.Group,
		IsPrivilege: s.IsPrivilege,
		Target:      s.Target,
		PipelineId:  s.PipelineID,
		Os:          s.Os,
		Process:     s.Process,
		Timer:       s.Timer,
		Modules:     s.Modules,
		Addons:      s.Addons,
	}
}

func (s *Session) Update(req *clientpb.RegisterSession) {
	s.Name = req.RegisterData.Name
	s.ProxyURL = req.RegisterData.Proxy
	s.Timer = req.RegisterData.Timer
	s.SessionContext.Update(req)

	if req.RegisterData.Sysinfo != nil {
		if !s.Initialized {
			s.Publish(consts.CtrlSessionInit, fmt.Sprintf("session %s init", s.ID))
		}
		s.UpdateSysInfo(req.RegisterData.Sysinfo)
	}
}

func (s *Session) UpdateSysInfo(info *implantpb.SysInfo) {
	s.Initialized = true
	info.Os.Name = strings.ToLower(info.Os.Name)
	if info.Os.Name == "windows" {
		info.Os.Arch = intermediate.FormatArch(info.Os.Arch)
	}
	s.IsPrivilege = info.IsPrivilege
	s.Filepath = info.Filepath
	s.WordDir = info.Workdir
	s.Os = info.Os
	s.Process = info.Process
}

func (s *Session) Publish(Op string, msg string) {
	EventBroker.Publish(Event{
		EventType: consts.EventSession,
		Op:        Op,
		Session:   s.ToProtobuf(),
		IsNotify:  true,
		Message:   msg,
	})
}

func (s *Session) nextTaskId() uint32 {
	s.taskseq++
	return s.taskseq
}

func (s *Session) SetLastTaskId(id uint32) {
	s.taskseq = id
}

func (s *Session) NewTask(name string, total int) *Task {
	task := &Task{
		Type:      name,
		Total:     total,
		Id:        s.nextTaskId(),
		SessionId: s.ID,
		Session:   s,
		DoneCh:    make(chan bool, total),
	}
	task.Ctx, task.Cancel = context.WithCancel(context.Background())
	s.Tasks.Add(task)
	go task.Handler()
	return task
}

func (s *Session) AllTask() []*Task {
	return s.Tasks.All()
}

func (s *Session) UpdateLastCheckin() {
	s.LastCheckin = uint64(time.Now().Unix())
}

// Request
func (s *Session) Request(msg *clientpb.SpiteRequest, stream grpc.ServerStream, timeout time.Duration) error {
	var err error
	done := make(chan struct{})
	go func() {
		err = stream.SendMsg(msg)
		if err != nil {
			logs.Log.Debugf(err.Error())
			return
		}
		close(done)
	}()
	select {
	case <-done:
		if err != nil {
			return err
		}
		return nil
	case <-time.After(timeout):
		return ErrImplantSendTimeout
	}
}

func (s *Session) RequestAndWait(msg *clientpb.SpiteRequest, stream grpc.ServerStream, timeout time.Duration) (*implantpb.Spite, error) {
	ch := make(chan *implantpb.Spite)
	s.StoreResp(msg.Task.TaskId, ch)
	err := s.Request(msg, stream, timeout)
	if err != nil {
		return nil, err
	}
	resp := <-ch
	// todo save to database
	return resp, nil
}

// RequestWithStream - 'async' means that the response is not returned immediately, but is returned through the channel 'ch
func (s *Session) RequestWithStream(msg *clientpb.SpiteRequest, stream grpc.ServerStream, timeout time.Duration) (chan *implantpb.Spite, chan *implantpb.Spite, error) {
	respCh := make(chan *implantpb.Spite)
	s.StoreResp(msg.Task.TaskId, respCh)
	err := s.Request(msg, stream, timeout)
	if err != nil {
		return nil, nil, err
	}

	in := make(chan *implantpb.Spite)
	go func() {
		defer close(respCh)
		var c = 0
		for spite := range in {
			err := stream.SendMsg(&clientpb.SpiteRequest{
				Session: msg.Session,
				Task:    msg.Task,
				Spite:   spite,
			})
			if err != nil {
				logs.Log.Debugf(err.Error())
				return
			}
			logs.Log.Debugf("send message %s, %d", spite.Name, c)
			c++
		}
	}()
	return in, respCh, nil
}

func (s *Session) RequestWithAsync(msg *clientpb.SpiteRequest, stream grpc.ServerStream, timeout time.Duration) (chan *implantpb.Spite, error) {
	respCh := make(chan *implantpb.Spite)
	s.StoreResp(msg.Task.TaskId, respCh)
	err := s.Request(msg, stream, timeout)
	if err != nil {
		return nil, err
	}

	return respCh, nil
}

func (s *Session) StoreResp(taskId uint32, ch chan *implantpb.Spite) {
	s.responses.Store(taskId, ch)
}

func (s *Session) GetResp(taskId uint32) (chan *implantpb.Spite, bool) {
	msg, ok := s.responses.Load(taskId)
	if !ok {
		return nil, false
	}
	return msg.(chan *implantpb.Spite), true
}

func (s *Session) DeleteResp(taskId uint32) {
	ch, ok := s.GetResp(taskId)
	if ok {
		close(ch)
	}
	s.responses.Delete(taskId)
}

type sessions struct {
	active *sync.Map // map[uint32]*Session
}

// All - Return a list of all sessions
func (s *sessions) All() []*Session {
	all := []*Session{}
	s.active.Range(func(key, value interface{}) bool {
		all = append(all, value.(*Session))
		return true
	})
	return all
}

// Get - Get a session by ID
func (s *sessions) Get(sessionID string) (*Session, bool) {
	if val, ok := s.active.Load(sessionID); ok {
		return val.(*Session), true
	}
	return nil, false
}

func (s *sessions) Add(session *Session) *Session {
	s.active.Store(session.ID, session)
	//EventBroker.Publish(Event{
	//	EventType: consts.SessionOpenedEvent,
	//	Session:   session,
	//})
	return session
}

func (s *sessions) Remove(sessionID string) {
	val, ok := s.active.Load(sessionID)
	if !ok {
		return
	}
	parentSession := val.(*Session)
	//children := findAllChildrenByPeerID(parentSession.PeerID)
	s.active.Delete(parentSession.ID)
	//coreLog.Debugf("Removing %d children of session %d (%v)", len(children), parentSession.ID, children)
	//for _, child := range children {
	//	childSession, ok := s.active.LoadAndDelete(child.SessionID)
	//	if ok {
	//		PivotSessions.Delete(childSession.(*Session).Connection.ID)
	//		EventBroker.Publish(Event{
	//			EventType: consts.SessionClosedEvent,
	//			Session:   childSession.(*Session),
	//		})
	//	}
	//}

	//Remove the parent session
	//EventBroker.Publish(Event{
	//	EventType: consts.SessionClosedEvent,
	//	Session:   parentSession,
	//})
}
