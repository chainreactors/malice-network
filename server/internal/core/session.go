package core

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/gookit/config/v2"
	"google.golang.org/grpc"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	// Sessions - Manages implant connections
	Sessions = &sessions{
		active: &sync.Map{},
	}
	ExtensionModules = []string{consts.ModuleExecuteBof, consts.ModuleExecuteDll}
	// ErrUnknownMessageType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessageType = errors.New("unknown message type")

	// ErrImplantSendTimeout - The implant did not respond prior to timeout deadline
	ErrImplantSendTimeout = errors.New("implant timeout")
)

func NewSession(req *lispb.RegisterSession) *Session {
	sess := &Session{
		Name:       req.RegisterData.Name,
		ProxyURL:   req.RegisterData.Proxy,
		Modules:    req.RegisterData.Module,
		Extensions: req.RegisterData.Extension,
		ID:         req.SessionId,
		PipelineID: req.ListenerId,
		RemoteAddr: req.RemoteAddr,
		Timer:      req.RegisterData.Timer,
		Tasks:      &Tasks{active: &sync.Map{}},
		Cache:      NewCache(10*consts.KB, path.Join(configs.CachePath, req.SessionId+".gob")),
		responses:  &sync.Map{},
	}

	if req.RegisterData.Sysinfo != nil {
		sess.UpdateSysInfo(req.RegisterData.Sysinfo)
	}

	return sess
}

// Session - Represents a connection to an implant
type Session struct {
	PipelineID string
	ID         string
	Name       string
	Group      string
	RemoteAddr string
	Os         *implantpb.Os
	Process    *implantpb.Process
	Timer      *implantpb.Timer
	Filepath   string
	WordDir    string
	ProxyURL   string
	Modules    []string
	Extensions *implantpb.Extensions
	Locale     string
	Tasks      *Tasks // task manager
	taskseq    uint32
	*Cache
	responses *sync.Map
	log       *logs.Logger
}

func (s *Session) Logger() *logs.Logger {
	var err error
	if s.log == nil {
		if auditLevel := config.Int(consts.AuditLevel); auditLevel > 0 {
			s.log, err = logs.NewFileLogger(filepath.Join(configs.AuditPath, s.ID+".log"))
			if err == nil {
				s.log.SuffixFunc = func() string {
					return time.Now().Format("2006-01-02 15:04.05")
				}
				if auditLevel == 2 {
					s.log.SetLevel(logs.Debug)
				}
			}
		}
	}
	return s.log
}

func (s *Session) ToProtobuf() *clientpb.Session {
	currentTime := time.Now()
	timeDiff := currentTime.Unix() - int64(s.Timer.LastCheckin)
	isAlive := uint64(timeDiff) <= s.Timer.Interval*2
	return &clientpb.Session{
		SessionId:  s.ID,
		Note:       s.Name,
		GroupName:  s.Group,
		IsAlive:    isAlive,
		RemoteAddr: s.RemoteAddr,
		ListenerId: s.PipelineID,
		Os:         s.Os,
		Process:    s.Process,
		Timer:      s.Timer,
		Tasks:      s.Tasks.ToProtobuf(),
		Modules:    s.Modules,
		Extensions: s.Extensions,
	}
}

func (s *Session) Update(req *lispb.RegisterSession) {
	s.Name = req.RegisterData.Name
	s.ProxyURL = req.RegisterData.Proxy
	s.Modules = req.RegisterData.Module
	s.Extensions = req.RegisterData.Extension
	s.Timer = req.RegisterData.Timer
	if req.RegisterData.Sysinfo != nil {
		s.UpdateSysInfo(req.RegisterData.Sysinfo)
	}
}

func (s *Session) UpdateSysInfo(info *implantpb.SysInfo) {
	info.Os.Name = strings.ToLower(info.Os.Name)
	if info.Os.Name == "windows" {
		info.Os.Arch = consts.GetWindowsArch(info.Os.Arch)
	}
	s.Filepath = info.Filepath
	s.WordDir = info.Workdir
	s.Os = info.Os
	s.Process = info.Process
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
	s.Timer.LastCheckin = uint64(time.Now().Unix())
}

// Request
func (s *Session) Request(msg *lispb.SpiteSession, stream grpc.ServerStream, timeout time.Duration) error {
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

func (s *Session) RequestAndWait(msg *lispb.SpiteSession, stream grpc.ServerStream, timeout time.Duration) (*implantpb.Spite, error) {
	ch := make(chan *implantpb.Spite)
	s.StoreResp(msg.TaskId, ch)
	err := s.Request(msg, stream, timeout)
	if err != nil {
		return nil, err
	}
	resp := <-ch
	// todo save to database
	return resp, nil
}

// RequestWithStream - 'async' means that the response is not returned immediately, but is returned through the channel 'ch
func (s *Session) RequestWithStream(msg *lispb.SpiteSession, stream grpc.ServerStream, timeout time.Duration) (chan *implantpb.Spite, chan *implantpb.Spite, error) {
	respCh := make(chan *implantpb.Spite)
	s.StoreResp(msg.TaskId, respCh)
	err := s.Request(msg, stream, timeout)
	if err != nil {
		return nil, nil, err
	}

	in := make(chan *implantpb.Spite)
	go func() {
		defer close(respCh)
		var c = 0
		for spite := range in {
			err := stream.SendMsg(&lispb.SpiteSession{
				SessionId: s.ID,
				TaskId:    msg.TaskId,
				Spite:     spite,
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

func (s *Session) RequestWithAsync(msg *lispb.SpiteSession, stream grpc.ServerStream, timeout time.Duration) (chan *implantpb.Spite, error) {
	respCh := make(chan *implantpb.Spite)
	s.StoreResp(msg.TaskId, respCh)
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

// sessions - Manages the slivers, provides atomic access
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

// Add - Add a sliver to the hive (atomically)
func (s *sessions) Add(session *Session) *Session {
	s.active.Store(session.ID, session)
	//EventBroker.Publish(Event{
	//	EventType: consts.SessionOpenedEvent,
	//	Session:   session,
	//})
	return session
}

// Remove - Remove a sliver from the hive (atomically)
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
