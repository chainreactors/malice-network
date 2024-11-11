package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/content"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gookit/config/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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
	Sessions         *sessions
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
	_, err := GlobalTicker.Start(consts.DefaultCacheInterval, func() {
		for _, session := range newSessions.All() {
			currentTime := time.Now()
			timeDiff := currentTime.Unix() - int64(session.LastCheckin)
			isAlive := timeDiff <= int64(1+session.Interval)*3
			if !isAlive {
				newSessions.Remove(session.ID)
				EventBroker.Publish(Event{
					EventType: consts.EventSession,
					Op:        consts.CtrlSessionLeave,
					Session:   session.ToProtobuf(),
					IsNotify:  true,
					Message:   fmt.Sprintf("session %s from %s at %s has stoped ", session.ID, session.Target, session.PipelineID),
				})
				if err := db.Session().Model(session.ToModel()).Update("IsAlive", isAlive).Error; err != nil {
					logs.Log.Errorf(err.Error())
				}
			}
		}

	})
	if err != nil {
		logs.Log.Errorf("cannot start ticker, %s", err.Error())
	}
	Sessions = newSessions
	return newSessions
}

func RegisterSession(req *clientpb.RegisterSession) (*Session, error) {
	cache := NewCache(path.Join(configs.CachePath, req.SessionId))
	err := cache.Save()
	if err != nil {
		return nil, err
	}
	sess := &Session{
		Type:           req.Type,
		Name:           req.RegisterData.Name,
		Group:          "default",
		ID:             req.SessionId,
		RawID:          req.RawId,
		PipelineID:     req.PipelineId,
		Target:         req.Target,
		Tasks:          NewTasks(),
		LastCheckin:    time.Now().Unix(),
		SessionContext: content.NewSessionContext(req),
		Taskseq:        1,
		Cache:          cache,
		responses:      &sync.Map{},
	}
	logDir := filepath.Join(configs.LogPath, sess.ID)
	err = os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		logs.Log.Errorf("cannot create log directory %s, %s", logDir, err.Error())
	}
	if req.RegisterData.Sysinfo != nil {
		sess.UpdateSysInfo(req.RegisterData.Sysinfo)
	}

	return sess, nil
}

func RecoverSession(sess *clientpb.Session) (*Session, error) {
	cache := NewCache(path.Join(configs.CachePath, sess.SessionId))
	err := cache.Load()
	if err != nil {
		return nil, err
	}
	s := &Session{
		Type:        sess.Type,
		Name:        sess.Note,
		Group:       sess.GroupName,
		ID:          sess.SessionId,
		RawID:       sess.RawId,
		PipelineID:  sess.PipelineId,
		Target:      sess.Target,
		Initialized: sess.IsInitialized,
		LastCheckin: sess.LastCheckin,
		Tasks:       NewTasks(),
		SessionContext: &content.SessionContext{SessionInfo: &content.SessionInfo{
			Os:       sess.Os,
			Process:  sess.Process,
			Interval: sess.Timer.Interval,
			Jitter:   sess.Timer.Jitter,
		},
			Modules: sess.Modules,
			Addons:  sess.Addons,
		},
		Taskseq:   1,
		Cache:     cache,
		responses: &sync.Map{},
	}
	tasks, tid, err := db.FindTaskAndMaxTasksID(s.ID)
	if err != nil {
		return nil, err
	}
	s.Taskseq = tid
	for _, task := range tasks {
		taskPb := task.ToProtobuf()
		s.Tasks.Add(FromTaskProtobuf(taskPb))
	}
	err = s.Recover()
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Session - Represents a connection to an implant
type Session struct {
	Type        string
	PipelineID  string
	ListenerID  string
	ID          string
	RawID       uint32
	Name        string
	Group       string
	Target      string
	Initialized bool
	LastCheckin int64
	Tasks       *Tasks // task manager
	*content.SessionContext

	*Cache
	Taskseq   uint32
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

func (s *Session) TaskLog(task *Task, spite *implantpb.Spite) error {
	data, err := proto.Marshal(spite)
	if err != nil {
		return err
	}
	filePath := filepath.Join(configs.LogPath, s.ID, fmt.Sprintf("%d_%d", task.Id, task.Cur))
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (s *Session) Recover() error {
	all := s.Cache.GetAll()
	tasks, err := db.GetAllTask()
	if err != nil {
		return err
	}
	for _, task := range tasks.Tasks {
		if task.Cur < task.Total {
			for key, value := range all {
				cacheTaskID := strconv.FormatUint(uint64(value.TaskId), 10) + "_" + strconv.FormatUint(uint64(task.Cur), 10)
				if err != nil {
					continue
				}
				if cacheTaskID == key {
					ch := make(chan *implantpb.Spite)
					s.responses.Store(task, ch)
				}
			}
		}
	}

	return nil
}

func (s *Session) ToProtobuf() *clientpb.Session {
	var isAlive bool
	if s.Type != consts.BindPipeline {
		currentTime := time.Now()
		timeDiff := currentTime.Unix() - int64(s.LastCheckin)
		isAlive = timeDiff <= 1+int64(s.Interval)*3
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
		Timer:       &implantpb.Timer{Interval: s.Interval, Jitter: s.Jitter},
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
		Timer:       &implantpb.Timer{Interval: s.Interval, Jitter: s.Jitter},
		Modules:     s.Modules,
		Addons:      s.Addons,
	}
}

func (s *Session) ToModel() *models.Session {
	contextContent, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	sessPb := s.ToProtobuf()
	return &models.Session{
		SessionID:   s.ID,
		RawID:       s.RawID,
		CreatedAt:   time.Now(),
		Note:        s.Name,
		GroupName:   s.Group,
		Target:      s.Target,
		Initialized: s.Initialized,
		Type:        s.Type,
		IsPrivilege: s.IsPrivilege,
		PipelineID:  s.PipelineID,
		IsAlive:     true,
		Context:     string(contextContent),
		LastCheckin: s.LastCheckin,
		Interval:    s.Interval,
		Jitter:      s.Jitter,
		Os:          models.FromOsPb(sessPb.Os),
		Process:     models.FromProcessPb(sessPb.Process),
	}
}

func (s *Session) Update(req *clientpb.RegisterSession) {
	s.Name = req.RegisterData.Name
	s.ProxyURL = req.RegisterData.Proxy
	s.Interval = req.RegisterData.Timer.Interval
	s.Jitter = req.RegisterData.Timer.Jitter
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
		info.Os.Arch = consts.FormatArch(info.Os.Arch)
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
	s.Taskseq++
	return s.Taskseq
}

func (s *Session) SetLastTaskId(id uint32) {
	s.Taskseq = id
}

func (s *Session) NewTask(name string, total int) *Task {
	task := &Task{
		Type:      name,
		Total:     total,
		Id:        s.nextTaskId(),
		SessionId: s.ID,
		Session:   s,
		DoneCh:    make(chan bool),
	}
	task.Ctx, task.Cancel = context.WithCancel(context.Background())
	s.Tasks.Add(task)
	return task
}

func (s *Session) AllTask() []*Task {
	return s.Tasks.All()
}

func (s *Session) UpdateLastCheckin() {
	s.LastCheckin = time.Now().Unix()
}

// Request
func (s *Session) Request(msg *clientpb.SpiteRequest, stream grpc.ServerStream) error {
	err := stream.SendMsg(msg)
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) RequestAndWait(msg *clientpb.SpiteRequest, stream grpc.ServerStream, timeout time.Duration) (*implantpb.Spite, error) {
	ch := make(chan *implantpb.Spite, 16)
	s.StoreResp(msg.Task.TaskId, ch)
	err := s.Request(msg, stream)
	if err != nil {
		return nil, err
	}
	resp := <-ch
	return resp, nil
}

// RequestWithStream - 'async' means that the response is not returned immediately, but is returned through the channel 'ch
func (s *Session) RequestWithStream(msg *clientpb.SpiteRequest, stream grpc.ServerStream, timeout time.Duration) (chan *implantpb.Spite, chan *implantpb.Spite, error) {
	respCh := make(chan *implantpb.Spite, 16)
	s.StoreResp(msg.Task.TaskId, respCh)
	err := s.Request(msg, stream)
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
	respCh := make(chan *implantpb.Spite, 16)
	s.StoreResp(msg.Task.TaskId, respCh)
	err := s.Request(msg, stream)
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
