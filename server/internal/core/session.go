package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/gookit/config/v2"
	"github.com/gorhill/cronexpr"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
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
	ErrSpiteStreamClosed  = errors.New("spite stream writer closed")

	// DB function variables — swappable in tests for mocking
	sessionDBSave        = func(s *models.Session) error { return db.SaveSessionModel(s) }
	sessionDBGetArtifact = func(name string) (*models.Artifact, error) { return db.GetArtifactByName(name) }
	sessionDBGetProfile  = func(name string) (*models.Profile, error) { return db.GetProfileByName(name) }
)

func createSessionDirs(sessionID string) (string, error) {
	contextDir, err := fileutils.SafeJoin(configs.ContextPath, sessionID)
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(contextDir, consts.CachePath)
	downloadDir := filepath.Join(contextDir, consts.DownloadPath)
	keyLoggerDir := filepath.Join(contextDir, consts.KeyLoggerPath)
	screenShotDir := filepath.Join(contextDir, consts.ScreenShotPath)
	taskDir := filepath.Join(contextDir, consts.TaskPath)
	requestDir := filepath.Join(contextDir, consts.RequestPath)

	dirs := []string{cacheDir, downloadDir, keyLoggerDir, screenShotDir, taskDir, requestDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", fmt.Errorf("cannot create directory %s: %w", dir, err)
		}
	}

	return cacheDir, nil
}

func NewSessions() *sessions {
	newSessions := &sessions{
		active: &sync.Map{},
	}
	_, err := GlobalTicker.Start(consts.DefaultCacheInterval, func() {
		newSessions.SweepInactive()
	})
	if err != nil {
		logs.Log.Errorf("cannot start ticker, %s", err.Error())
	}
	Sessions = newSessions
	return newSessions
}

func RegisterSession(req *clientpb.RegisterSession) (*Session, error) {
	if req == nil || req.RegisterData == nil {
		return nil, types.ErrMissingRequestField
	}
	if req.SessionId == "" {
		return nil, types.ErrInvalidSessionID
	}
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
		SessionContext: client.NewSessionContext(req),
		Taskseq:        1,
		Cache:          cache,
		responses:      &sync.Map{},
	}

	// 从pipeline获取预分发的密钥对
	err = sess.initializeSecureManager(req)
	if err != nil {
		logs.Log.Errorf("[secure] failed to initialize pipeline keypair: %v", err)
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
	cachePath, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(sess.SessionID, consts.CachePath, CacheName))
	if err != nil {
		return nil, err
	}
	cache := NewCache(cachePath)
	err = cache.Load()
	if err != nil {
		return nil, err
	}

	sessionContext := sess.Data
	if sessionContext == nil && sess.DataString != "" {
		recovered, err := client.RecoverSessionContext(sess.DataString)
		if err != nil {
			logs.Log.Warnf("failed to recover session context %s: %v", sess.SessionID, err)
		} else {
			sessionContext = recovered
		}
	}
	if sessionContext == nil {
		sessionContext = &client.SessionContext{}
	}
	if sessionContext.SessionInfo == nil {
		sessionContext.SessionInfo = &client.SessionInfo{}
	}
	if sessionContext.Os == nil {
		sessionContext.Os = &implantpb.Os{}
	}
	if sessionContext.Process == nil {
		sessionContext.Process = &implantpb.Process{}
	}
	if sessionContext.Argue == nil {
		sessionContext.Argue = map[string]string{}
	}
	if sessionContext.Any == nil {
		sessionContext.Any = map[string]interface{}{}
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
		CreatedAt:      sess.CreatedAt,
		Tasks:          NewTasks(),
		SessionContext: sessionContext,
		Taskseq:        1,
		Cache:          cache,
		responses:      &sync.Map{},
	}

	// 无论如何都初始化 SecureManager，使用SessionContext中的KeyPair
	err = s.initializeSecureManager(&clientpb.RegisterSession{
		PipelineId:   sess.PipelineID,
		ListenerId:   sess.ListenerID,
		RegisterData: &implantpb.Register{Secure: s.Secure},
	})
	if err != nil {
		return nil, err
	}
	s.SetLastCheckin(sess.LastCheckin)

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
		recoverTask.Session = s
		recoverTask.DoneCh = make(chan bool, 1)
		recoverTask.Ctx, recoverTask.Cancel = context.WithCancel(s.Ctx)
		if recoverTask.Finished() {
			recoverTask.Closed = true
			close(recoverTask.DoneCh)
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
	CreatedAt   time.Time
	Tasks       *Tasks // task manager
	*client.SessionContext

	// Age 密码学安全管理器（运行时，负责密钥交换和轮换）
	SecureManager *SecureManager

	*Cache
	Taskseq   uint32
	responses *sync.Map
	rpcLog    *logs.Logger

	lastCheckin atomic.Int64
	deadState   atomic.Bool

	keepaliveMu      sync.Mutex
	keepaliveEnabled bool
	anyMu            sync.RWMutex

	Ctx    context.Context
	Cancel context.CancelFunc
}

type SpiteStreamWriter struct {
	messages  chan *implantpb.Spite
	done      chan struct{}
	closeOnce sync.Once
	doneOnce  sync.Once
	errMu     sync.RWMutex
	err       error
}

func newSpiteStreamWriter(buffer int) *SpiteStreamWriter {
	return &SpiteStreamWriter{
		messages: make(chan *implantpb.Spite, buffer),
		done:     make(chan struct{}),
	}
}

func (w *SpiteStreamWriter) Send(spite *implantpb.Spite) error {
	if spite == nil {
		return errors.New("spite is nil")
	}

	select {
	case <-w.done:
		return w.Err()
	default:
	}

	select {
	case <-w.done:
		return w.Err()
	case w.messages <- spite:
		return nil
	}
}

func (w *SpiteStreamWriter) Close() {
	if w == nil {
		return
	}
	w.closeOnce.Do(func() {
		close(w.messages)
	})
}

func (w *SpiteStreamWriter) Err() error {
	if w == nil {
		return ErrSpiteStreamClosed
	}
	w.errMu.RLock()
	if w.err != nil {
		defer w.errMu.RUnlock()
		return w.err
	}
	w.errMu.RUnlock()
	select {
	case <-w.done:
		return ErrSpiteStreamClosed
	default:
		return nil
	}
}

func (w *SpiteStreamWriter) finish(err error) {
	if w == nil {
		return
	}
	w.doneOnce.Do(func() {
		if err != nil {
			w.errMu.Lock()
			w.err = err
			w.errMu.Unlock()
		}
		close(w.done)
	})
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
	cur, _ := task.Progress()
	index := cur
	if index > 0 {
		index--
	}
	// Task cache uses zero-based callback indexes. Persist the same index so
	// WaitTaskContent/GetTaskContent do not return the wrong callback when they
	// fall back from memory to disk.
	filePath, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(s.ID, consts.TaskPath, fmt.Sprintf("%d_%d", task.Id, index)))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (s *Session) Recover() error {
	for _, task := range s.Tasks.All() {
		cur, total := task.Progress()
		if cur < total {
			ch := make(chan *implantpb.Spite, 16)
			s.responses.Store(task.Id, ch)
		}
	}
	return nil
}

func (s *Session) RecoverTaskIDByLog() (int, error) {
	taskDir, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(s.ID, consts.TaskPath))
	if err != nil {
		return 0, err
	}
	files, err := os.ReadDir(taskDir)
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

func (s *Session) SetLastCheckin(ts int64) {
	s.lastCheckin.Store(ts)
}

func (s *Session) MarkDead() bool {
	return s.deadState.CompareAndSwap(false, true)
}

func (s *Session) MarkAlive() bool {
	return s.deadState.CompareAndSwap(true, false)
}

func (s *Session) IsMarkedDead() bool {
	return s.deadState.Load()
}

func (s *Session) HasUnfinishedTasks() bool {
	if s.Tasks == nil {
		return false
	}
	for _, task := range s.Tasks.All() {
		if task != nil && !task.Finished() {
			return true
		}
	}
	return false
}

func (s *Session) LastCheckinUnix() int64 {
	return s.lastCheckin.Load()
}

func (s *Session) isAlived() bool {
	if s == nil {
		return false
	}
	if s.Type == consts.BindPipeline {
		return true
	} else {
		parsedExpr, err := cronexpr.Parse(s.Expression)
		if err != nil {
			logs.Errorf("exp parse error %s", err)
			return true
		}
		nextTime := parsedExpr.Next(time.Now())
		remainingSeconds := int64(nextTime.Sub(time.Now()).Seconds())
		remainingSeconds = int64(float64(remainingSeconds) * (1 + s.Jitter))
		allowedOffline := utils.Max(remainingSeconds+30, int64(90)) // values are in seconds
		return time.Now().Unix()-s.LastCheckinUnix() <= allowedOffline
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
		LastCheckin: s.LastCheckinUnix(),
		Filepath:    s.SessionContext.Filepath,
		Workdir:     s.SessionContext.WorkDir,
		Locate:      s.SessionContext.Locale,
		Proxy:       s.SessionContext.ProxyURL,
		Timer:       &implantpb.Timer{Expression: s.Expression, Jitter: s.Jitter},
		CreatedAt:   s.CreatedAt.Unix(),
		Tasks:       s.Tasks.ToProtobuf(),
		Modules:     s.Modules,
		Addons:      s.Addons,
		KeyPair:     s.KeyPair, // 添加密钥对
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
		IsAlive:     s.isAlived(),
		IsPrivilege: s.IsPrivilege,
		Target:      s.Target,
		PipelineId:  s.PipelineID,
		ListenerId:  s.ListenerID,
		Os:          s.Os,
		Process:     s.Process,
		LastCheckin: s.LastCheckinUnix(),
		Filepath:    s.SessionContext.Filepath,
		Workdir:     s.SessionContext.WorkDir,
		Locate:      s.SessionContext.Locale,
		Proxy:       s.SessionContext.ProxyURL,
		Timer:       &implantpb.Timer{Expression: s.Expression, Jitter: s.Jitter},
		CreatedAt:   s.CreatedAt.Unix(),
		Modules:     s.Modules,
		Addons:      s.Addons,
		KeyPair:     s.KeyPair,
		Data:        s.Marshal(),
	}
}

func (s *Session) Save() error {
	return sessionDBSave(s.ToModel())
}

func (s *Session) ToModel() *models.Session {
	sessModel := &models.Session{
		SessionID:   s.ID,
		RawID:       s.RawID,
		Note:        s.Note,
		GroupName:   s.Group,
		Target:      s.Target,
		Initialized: s.Initialized,
		Type:        s.Type,
		PipelineID:  s.PipelineID,
		ListenerID:  s.ListenerID,
		IsAlive:     s.isAlived(),
		LastCheckin: s.LastCheckinUnix(),
		DataString:  s.Marshal(),
	}
	artifact, err := sessionDBGetArtifact(s.Name)
	if err == nil && artifact.ProfileName != "" {
		if _, profileErr := sessionDBGetProfile(artifact.ProfileName); profileErr == nil {
			sessModel.ProfileName = artifact.ProfileName
		}
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

// SaveAndNotify 持久化 session 到数据库并广播更新事件到所有客户端。
// 这是 RPC handler 修改 session 持久化字段后的标准路径。
func (s *Session) SaveAndNotify(msg string) error {
	if err := s.Save(); err != nil {
		logs.Log.Errorf("save session %s failed: %s", s.ID, err.Error())
		return err
	}
	s.PushUpdate(msg)
	return nil
}

// GetPipelineEncryptionKey returns the transport encryption key from the
// session's pipeline config. Returns "" if not found.
func (s *Session) GetPipelineEncryptionKey() string {
	pipeline, ok := Listeners.Find(s.PipelineID)
	if !ok || pipeline == nil {
		return ""
	}
	encryptions := pipeline.GetEncryption()
	if len(encryptions) == 0 {
		return ""
	}
	return encryptions[0].Key
}

// GetPacketLength returns the per-pipeline packet length or falls back to global config.
func (s *Session) GetPacketLength() int {
	pipeline, ok := Listeners.Find(s.PipelineID)
	if ok && pipeline != nil && pipeline.PacketLength > 0 {
		return int(pipeline.PacketLength)
	}
	return config.Int(consts.ConfigMaxPacketLength)
}

func (s *Session) Update(req *clientpb.RegisterSession) {
	s.Name = req.RegisterData.Name
	s.PipelineID = req.PipelineId
	s.ListenerID = req.ListenerId
	s.ProxyURL = req.RegisterData.Proxy
	if req.RegisterData.Timer != nil {
		s.Expression = req.RegisterData.Timer.Expression
		s.Jitter = req.RegisterData.Timer.Jitter
	}
	s.SessionContext.Update(req)

	// SecureManager现在使用固定的100次交互计数，不需要更新间隔

	if req.RegisterData.Sysinfo != nil {
		if !s.Initialized {
			s.Publish(consts.CtrlSessionInit, fmt.Sprintf("session %s init", s.ID), true, true)
		}
		s.UpdateSysInfo(req.RegisterData.Sysinfo)
	}

	err := s.Save()
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
	artifact, err := sessionDBGetArtifact(s.Name)
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
	now := time.Now()
	task := &Task{
		Type:      name,
		Total:     total,
		Id:        s.Taskseq,
		SessionId: s.ID,
		Session:   s,
		DoneCh:    make(chan bool, 1),
		CreatedAt: now,
		Deadline:  now.Add(consts.MinTimeout),
	}
	task.Ctx, task.Cancel = context.WithCancel(s.Ctx)
	s.Tasks.Add(task)
	return task
}

// Request
func (s *Session) Request(msg *clientpb.SpiteRequest, stream grpc.ServerStream) error {
	return stream.SendMsg(msg)
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
func (s *Session) RequestWithStream(msg *clientpb.SpiteRequest, stream grpc.ServerStream, timeout time.Duration) (*SpiteStreamWriter, chan *implantpb.Spite, error) {
	respCh := make(chan *implantpb.Spite, 16)
	s.StoreResp(msg.Task.TaskId, respCh)
	err := s.Request(msg, stream)
	if err != nil {
		return nil, nil, err
	}

	writer := newSpiteStreamWriter(1)
	GoGuarded("session-request-stream:"+s.ID, func() error {
		defer close(respCh)
		defer writer.finish(nil)
		var c = 0
		for spite := range writer.messages {
			sendErr := stream.SendMsg(&clientpb.SpiteRequest{
				Session: msg.Session,
				Task:    msg.Task,
				Spite:   spite,
			})
			if sendErr != nil {
				err = fmt.Errorf("session %s stream send failed for task %d: %w", s.ID, msg.Task.TaskId, sendErr)
				writer.finish(err)
				return err
			}
			logs.Log.Debugf("send message %s, %d", spite.Name, c)
			c++
		}
		return nil
	}, LogGuardedError("session-request-stream:"+s.ID))
	return writer, respCh, nil
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

func (s *Session) SetAny(id string, value interface{}) {
	s.anyMu.Lock()
	defer s.anyMu.Unlock()
	if s.SessionContext == nil {
		s.SessionContext = &client.SessionContext{}
	}
	if s.SessionContext.Any == nil {
		s.SessionContext.Any = make(map[string]interface{})
	}
	s.SessionContext.Any[id] = value
}

func (s *Session) GetAny(id string) (interface{}, bool) {
	s.anyMu.RLock()
	defer s.anyMu.RUnlock()
	if s.SessionContext == nil || s.SessionContext.Any == nil {
		return nil, false
	}
	value, ok := s.SessionContext.Any[id]
	return value, ok
}

func (s *Session) DeleteAny(id string) {
	s.anyMu.Lock()
	defer s.anyMu.Unlock()
	if s.SessionContext == nil || s.SessionContext.Any == nil {
		return
	}
	delete(s.SessionContext.Any, id)
}

func (s *Session) GetResp(taskId uint32) (chan *implantpb.Spite, bool) {
	msg, ok := s.responses.Load(taskId)
	if !ok {
		return nil, false
	}
	return msg.(chan *implantpb.Spite), true
}

// RemoveResp removes the response channel from the map without closing it.
// This prevents new producers from finding the channel, while existing
// producer goroutines that already hold a reference can still safely send
// into the buffer without panicking on a closed channel.
func (s *Session) RemoveResp(taskId uint32) {
	s.responses.Delete(taskId)
}

func (s *Session) DeleteResp(taskId uint32) {
	ch, ok := s.GetResp(taskId)
	if ok {
		close(ch)
	}
	s.responses.Delete(taskId)
}

// UpdateKeyPair 更新KeyPair并同步到SecureManager
func (s *Session) UpdateKeyPair(keyPair *clientpb.KeyPair) {
	s.SessionContext.KeyPair = keyPair
	// 更新SecureManager中的KeyPair引用
	if s.SecureManager != nil {
		s.SecureManager.UpdateKeyPair(keyPair)
	}
}

// SetKeepalive updates the keepalive state. Returns the previous state.
func (s *Session) SetKeepalive(enabled bool) bool {
	s.keepaliveMu.Lock()
	defer s.keepaliveMu.Unlock()
	prev := s.keepaliveEnabled
	s.keepaliveEnabled = enabled
	if prev != enabled {
		logs.Log.Infof("[keepalive] session %s: %v -> %v", s.ID, prev, enabled)
	}
	return prev
}

// IsKeepaliveEnabled returns the current keepalive status.
func (s *Session) IsKeepaliveEnabled() bool {
	s.keepaliveMu.Lock()
	defer s.keepaliveMu.Unlock()
	return s.keepaliveEnabled
}

// ResetKeepalive resets keepalive state (used on disconnect).
func (s *Session) ResetKeepalive() {
	s.keepaliveMu.Lock()
	defer s.keepaliveMu.Unlock()
	s.keepaliveEnabled = false
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
	return nil, types.ErrNotFoundSession
}

func (s *sessions) Add(session *Session) *Session {
	if session == nil {
		logs.Log.Errorf("session is nil")
		return nil
	}
	session.MarkAlive()
	s.active.Store(session.ID, session)
	return session
}

func (s *sessions) Remove(sessionID string) {
	val, ok := s.active.Load(sessionID)
	if !ok {
		return
	}
	parentSession := val.(*Session)
	parentSession.ResetKeepalive()
	parentSession.Cancel()
	s.active.Delete(parentSession.ID)
}

func SweepInactiveSessions() {
	if Sessions == nil {
		return
	}
	Sessions.SweepInactive()
}

func (s *sessions) SweepInactive() {
	if s == nil {
		return
	}
	for _, session := range s.All() {
		if session == nil || session.isAlived() {
			continue
		}
		if err := session.Save(); err != nil {
			logs.Log.Errorf("save dead session %s failed: %s", session.ID, err.Error())
		}
		if session.MarkDead() {
			session.Publish(
				consts.CtrlSessionDead,
				fmt.Sprintf("session %s from %s at %s may have left ", session.ID, session.Target, session.PipelineID),
				true,
				true,
			)
		}
		if session.HasUnfinishedTasks() {
			continue
		}
		s.Remove(session.ID)
	}
}

// initializePipelineKeyPair 从pipeline获取预分发的密钥对
func (s *Session) initializeSecureManager(req *clientpb.RegisterSession) error {
	var (
		pipeline *clientpb.Pipeline
		ok       bool
	)

	if req.ListenerId != "" {
		pipeline, ok = Listeners.FindByListener(req.ListenerId, req.PipelineId)
	} else if s.ListenerID != "" {
		pipeline, ok = Listeners.FindByListener(s.ListenerID, req.PipelineId)
	} else {
		pipeline, ok = Listeners.Find(req.PipelineId)
	}

	if !ok {
		return fmt.Errorf("failed to get pipeline")
	}

	if pipeline == nil || pipeline.Secure == nil || !pipeline.Secure.Enable {
		logs.Log.Debugf("[secure] pipeline secure mode not enabled for session %s", s.ID)
		return nil
	}

	if req.RegisterData.Secure == nil || !req.RegisterData.Secure.Enable {
		logs.Log.Debugf("[secure] session secure mode enabled for session %s", s.ID)
		return nil
	}

	if s.KeyPair == nil {
		s.KeyPair = &clientpb.KeyPair{
			PublicKey:  pipeline.Secure.ImplantKeypair.PublicKey,
			PrivateKey: pipeline.Secure.ServerKeypair.PrivateKey,
		}
	}

	s.PushCtrl()
	logs.Log.Infof("[secure] initialized session %s", s.ID)

	s.SecureManager = NewSecureSpiteManager(s)
	return nil
}

func (s *Session) UpdatePublicKey(key string) {
	s.UpdateKeyPairFieldsAndPushCtrl(key, "")
}

func (s *Session) UpdatePrivateKey(key string) {
	s.UpdateKeyPairFieldsAndPushCtrl("", key)
}

func (s *Session) UpdateKeyPairFieldsAndPushCtrl(publicKey string, privateKey string) {
	next := &clientpb.KeyPair{}
	if s.KeyPair != nil {
		next.PublicKey = s.KeyPair.PublicKey
		next.PrivateKey = s.KeyPair.PrivateKey
	}
	if publicKey != "" {
		next.PublicKey = publicKey
	}
	if privateKey != "" {
		next.PrivateKey = privateKey
	}
	s.UpdateKeyPair(next)
	s.PushCtrl()
}

func (s *Session) PushCtrl() {
	lns, err := Listeners.Get(s.ListenerID)
	if err != nil {
		return
	}
	s.Save()
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl:    consts.CtrlListenerSyncSession,
		Session: s.ToProtobufLite(),
	})
}
