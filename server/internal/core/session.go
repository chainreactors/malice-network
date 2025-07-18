package core

import (
	"context"
	"errors"
	"fmt"
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

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
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

func createSessionDirs(sessionID string) (string, error) {
	contextDir := filepath.Join(configs.ContextPath, sessionID)
	cacheDir := filepath.Join(contextDir, consts.CachePath)
	downloadDir := filepath.Join(contextDir, consts.DownloadPath)
	keyLoggerDir := filepath.Join(contextDir, consts.KeyLoggerPath)
	screenShotDir := filepath.Join(contextDir, consts.ScreenShotPath)
	taskDir := filepath.Join(contextDir, consts.TaskPath)
	requestDir := filepath.Join(contextDir, consts.RequestPath)

	dirs := []string{cacheDir, downloadDir, keyLoggerDir, screenShotDir, taskDir, requestDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			logs.Log.Errorf("cannot create directory %s, %s", dir, err.Error())
		}
	}

	return cacheDir, nil
}

func NewSessions() *sessions {
	newSessions := &sessions{
		active: &sync.Map{},
	}
	_, err := GlobalTicker.Start(consts.DefaultCacheInterval, func() {
		for _, session := range newSessions.All() {
			sessModel := session.ToModel()
			if !session.isAlived() {
				sessModel.IsAlive = false
				session.Publish(consts.CtrlSessionDead, fmt.Sprintf("session %s from %s at %s has leaved ", session.ID, session.Target, session.PipelineID), true, true)
				//newSessions.Remove(session.ID)
			}
			err := db.Session().Save(sessModel).Error
			if err != nil {
				logs.Log.Errorf("update session %s info failed in db, %s", session.ID, err.Error())
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
	current_time := time.Now().Unix()
	cacheDir, err := createSessionDirs(req.SessionId)
	if err != nil {
		return nil, err
	}
	cache := NewCache(filepath.Join(cacheDir, CacheName))
	err = cache.Save()
	if err != nil {
		return nil, err
	}
	sess := &Session{
		Type:           req.Type,
		Name:           req.RegisterData.Name,
		Group:          "default",
		Note:           req.RegisterData.Name,
		ID:             req.SessionId,
		RawID:          req.RawId,
		PipelineID:     req.PipelineId,
		ListenerID:     req.ListenerId,
		Target:         req.Target,
		Tasks:          NewTasks(),
		CreatedAt:      time.Unix(current_time, 0),
		SessionContext: types.NewSessionContext(req),
		Taskseq:        1,
		Cache:          cache,
		responses:      &sync.Map{},
	}
	sess.Ctx, sess.Cancel = context.WithCancel(context.Background())
	if req.RegisterData.Sysinfo != nil {
		sess.UpdateSysInfo(req.RegisterData.Sysinfo)
	} else {
		sess.FillSysInfo()
	}

	return sess, nil
}

func RecoverSession(sess *models.Session) (*Session, error) {
	cache := NewCache(path.Join(configs.ContextPath, sess.SessionID, consts.CachePath, CacheName))
	err := cache.Load()
	if err != nil {
		return nil, err
	}
	s := &Session{
		Type:           sess.Type,
		Name:           sess.ProfileName,
		Note:           sess.Note,
		Group:          sess.GroupName,
		ID:             sess.SessionID,
		RawID:          sess.RawID,
		PipelineID:     sess.PipelineID,
		ListenerID:     sess.ListenerID,
		Target:         sess.Target,
		Initialized:    sess.Initialized,
		LastCheckin:    sess.LastCheckin,
		CreatedAt:      sess.CreatedAt,
		Tasks:          NewTasks(),
		SessionContext: sess.Data,
		Taskseq:        1,
		Cache:          cache,
		responses:      &sync.Map{},
	}
	s.Ctx, s.Cancel = context.WithCancel(context.Background())
	tasks, tid, err := db.FindTaskAndMaxTasksID(s.ID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		logID, err := s.RecoverTaskIDByLog()
		if err != nil {
			return nil, err
		}
		if uint32(logID) > tid {
			tid = uint32(logID)
		}
	}
	s.Taskseq = tid
	for _, task := range tasks {
		taskPb := task.ToProtobuf()
		recoverTask := FromTaskProtobuf(taskPb)
		recoverTask.Ctx, recoverTask.Cancel = context.WithCancel(s.Ctx)
		if recoverTask.Total == recoverTask.Cur {
			recoverTask.Cancel()
		}
		s.Tasks.Add(recoverTask)
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
	Note        string
	Target      string
	Initialized bool
	LastCheckin int64
	CreatedAt   time.Time
	Tasks       *Tasks // task manager
	*types.SessionContext

	*Cache
	Taskseq   uint32
	responses *sync.Map
	rpcLog    *logs.Logger

	Ctx    context.Context
	Cancel context.CancelFunc
}

func (s *Session) Abstract() string {
	if s.Os == nil {
		return fmt.Sprintf("%s(%s)", s.Name, s.ID)
	} else {
		if s.IsPrivilege {
			return fmt.Sprintf("%s(%s) %s/%s %s *", s.Name, s.ID, s.Os.Name, s.Os.Arch, s.Os.Username)
		}
		return fmt.Sprintf("%s(%s) %s/%s %s", s.Name, s.ID, s.Os.Name, s.Os.Arch, s.Os.Username)
	}
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
					s.rpcLog.SetLevel(logs.DebugLevel)
					s.rpcLog.PrefixFunc = func() string {
						return ""
					}
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
	filePath := filepath.Join(configs.ContextPath, s.ID, consts.TaskPath, fmt.Sprintf("%d_%d", task.Id, task.Cur))
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (s *Session) Recover() error {
	tasks, err := db.GetAllTask()
	if err != nil {
		return err
	}
	for _, task := range tasks.Tasks {
		if task.Cur < task.Total {
			ch := make(chan *implantpb.Spite, 16)
			s.responses.Store(task, ch)
		}
	}

	return nil
}

func (s *Session) RecoverTaskIDByLog() (int, error) {
	files, err := os.ReadDir(filepath.Join(configs.ContextPath, s.ID, consts.TaskPath))
	if err != nil {
		return 0, err
	}

	maxTaskID := 0

	for _, file := range files {
		parts := strings.Split(file.Name(), "_")
		if len(parts) < 2 {
			continue
		}

		taskID, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if taskID > maxTaskID {
			maxTaskID = taskID
		}
	}

	return maxTaskID, nil
}

func (s *Session) isAlived() bool {
	if s.Type == consts.BindPipeline {
		return true
	} else {
		return time.Now().Unix()-s.LastCheckin <= utils.Max((1+int64(s.Interval))*10, int64(time.Second*60))
	}
}

func (s *Session) ToProtobuf() *clientpb.Session {
	return &clientpb.Session{
		Type:        s.Type,
		SessionId:   s.ID,
		RawId:       s.RawID,
		Note:        s.Note,
		Name:        s.Name,
		GroupName:   s.Group,
		IsAlive:     s.isAlived(),
		IsPrivilege: s.IsPrivilege,
		Target:      s.Target,
		PipelineId:  s.PipelineID,
		ListenerId:  s.ListenerID,
		Os:          s.Os,
		Process:     s.Process,
		LastCheckin: s.LastCheckin,
		Filepath:    s.SessionContext.Filepath,
		Workdir:     s.SessionContext.WorkDir,
		Locate:      s.SessionContext.Locale,
		Proxy:       s.SessionContext.ProxyURL,
		Timer:       &implantpb.Timer{Interval: s.Interval, Jitter: s.Jitter},
		CreatedAt:   s.CreatedAt.Unix(),
		Tasks:       s.Tasks.ToProtobuf(),
		Modules:     s.Modules,
		Addons:      s.Addons,
		Data:        s.Marshal(),
	}
}

func (s *Session) ToProtobufLite() *clientpb.Session {
	return &clientpb.Session{
		Type:        s.Type,
		SessionId:   s.ID,
		RawId:       s.RawID,
		Note:        s.Note,
		Name:        s.Name,
		GroupName:   s.Group,
		IsPrivilege: s.IsPrivilege,
		Target:      s.Target,
		PipelineId:  s.PipelineID,
		ListenerId:  s.ListenerID,
		Os:          s.Os,
		Process:     s.Process,
		LastCheckin: s.LastCheckin,
		Filepath:    s.SessionContext.Filepath,
		Workdir:     s.SessionContext.WorkDir,
		Locate:      s.SessionContext.Locale,
		Proxy:       s.SessionContext.ProxyURL,
		Timer:       &implantpb.Timer{Interval: s.Interval, Jitter: s.Jitter},
		Modules:     s.Modules,
		Addons:      s.Addons,
		Data:        s.Marshal(),
	}
}

func (s *Session) ToModel() *models.Session {
	sessModel := &models.Session{
		SessionID:   s.ID,
		RawID:       s.RawID,
		Note:        s.Note,
		ProfileName: s.Name,
		GroupName:   s.Group,
		Target:      s.Target,
		Initialized: s.Initialized,
		Type:        s.Type,
		PipelineID:  s.PipelineID,
		ListenerID:  s.ListenerID,
		IsAlive:     true,
		LastCheckin: s.LastCheckin,
		DataString:  s.Marshal(),
	}
	artifact, err := db.GetArtifactByName(s.Note)
	if err == nil {
		sessModel.ProfileName = artifact.Name
	}
	return sessModel
}

func (s *Session) PushUpdate(msg string) {
	EventBroker.Publish(Event{
		EventType: consts.EventSession,
		Op:        consts.CtrlSessionUpdate,
		Session:   s.ToProtobufLite(),
		Message:   msg,
	})
}

func (s *Session) Update(req *clientpb.RegisterSession) {
	s.Name = req.RegisterData.Name
	s.PipelineID = req.PipelineId
	s.ListenerID = req.ListenerId
	s.ProxyURL = req.RegisterData.Proxy
	s.Interval = req.RegisterData.Timer.Interval
	s.Jitter = req.RegisterData.Timer.Jitter
	s.SessionContext.Update(req)

	if req.RegisterData.Sysinfo != nil {
		if !s.Initialized {
			s.Publish(consts.CtrlSessionInit, fmt.Sprintf("session %s init", s.ID), true, true)
		}
		s.UpdateSysInfo(req.RegisterData.Sysinfo)
	}

	err := db.Session().Save(s.ToModel()).Error
	if err != nil {
		logs.Log.Errorf("update session %s info failed in db, %s", s.ID, err.Error())
	}
}

func (s *Session) UpdateSysInfo(info *implantpb.SysInfo) {
	s.Initialized = true
	info.Os.Name = strings.ToLower(info.Os.Name)
	info.Os.Arch = consts.FormatArch(info.Os.Arch)
	s.IsPrivilege = info.IsPrivilege
	s.Filepath = info.Filepath
	s.WorkDir = info.Workdir
	s.Os = info.Os
	s.Process = info.Process
}

func (s *Session) FillSysInfo() {
	artifact, err := db.GetArtifactByName(s.Name)
	if err != nil {
		logs.Log.Errorf("failed to find atrtifact %s: %s", s.Name, err)
		return
	}
	s.Os = &implantpb.Os{
		Name: artifact.Os,
		Arch: artifact.Arch,
	}
}

func (s *Session) Publish(Op string, msg string, notify bool, important bool) {
	EventBroker.Publish(Event{
		EventType: consts.EventSession,
		Op:        Op,
		Session:   s.ToProtobufLite(),
		IsNotify:  notify,
		Message:   msg,
		Important: important,
	})
}

func (s *Session) NewTask(name string, total int) *Task {
	s.Taskseq++
	task := &Task{
		Type:      name,
		Total:     total,
		Id:        s.Taskseq,
		SessionId: s.ID,
		Session:   s,
		DoneCh:    make(chan bool),
	}
	task.Ctx, task.Cancel = context.WithCancel(s.Ctx)
	s.Tasks.Add(task)
	return task
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
func (s *sessions) Get(sessionID string) (*Session, error) {
	if val, ok := s.active.Load(sessionID); ok {
		return val.(*Session), nil
	}
	return nil, errs.ErrNotFoundSession
}

func (s *sessions) Add(session *Session) *Session {
	if session == nil {
		logs.Log.Errorf("session is nil")
		return nil
	}
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
