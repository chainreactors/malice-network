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

	ErrSpiteStreamClosed = errors.New("spite stream writer closed")

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
		cache.Close()
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
		Cache:          cache,
		responses:      &sync.Map{},
	}
	sess.Taskseq.Store(1)

	// 从pipeline获取预分发的密钥对
	err = sess.initializeSecureManager(req)
	if err != nil {
		logs.Log.Errorf("secure - init_pipeline_keypair_failed error=%q", err)
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
	cache := loadRecoveredCache(cachePath, sess.SessionID)
	recovered := false
	defer func() {
		if !recovered {
			cache.Close()
		}
	}()

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
		Cache:          cache,
		responses:      &sync.Map{},
	}
	s.Taskseq.Store(1)

	s.restoreSecureManagerFromContext()
	s.SetLastCheckin(sess.LastCheckin)

	s.Ctx, s.Cancel = context.WithCancel(context.Background())
	tasks, tid, err := db.FindTaskAndMaxTasksID(s.ID)
	if err != nil {
		return nil, err
	}
	logID, err := s.RecoverTaskIDByLog()
	if err != nil {
		return nil, err
	}
	if uint32(logID) > tid {
		tid = uint32(logID)
	}
	s.Taskseq.Store(tid)
	for _, task := range tasks {
		taskPb := task.ToProtobuf()
		recoverTask := FromTaskProtobuf(taskPb)
		recoverTask.Session = s
		recoverTask.DoneCh = make(chan bool, 1)
		recoverTask.Ctx, recoverTask.Cancel = context.WithCancel(s.Ctx)
		if recoverTask.Finished() {
			recoverTask.closed.Store(true)
			close(recoverTask.DoneCh)
			recoverTask.Cancel()
		}
		s.Tasks.Add(recoverTask)
	}
	err = s.Recover()
	if err != nil {
		return nil, err
	}
	recovered = true
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
	Taskseq   atomic.Uint32
	responses *sync.Map
	rpcLog    *logs.Logger
	rpcLogMu  sync.Mutex

	lastCheckin atomic.Int64
	deadState   atomic.Bool

	keepaliveMu      sync.Mutex
	keepaliveEnabled bool
	anyMu            sync.RWMutex
	stateMu          sync.RWMutex

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
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
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
	s.rpcLogMu.Lock()
	defer s.rpcLogMu.Unlock()
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
		if !task.Finished() {
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
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
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
			logs.Log.Debugf("skip non-numeric task log filename: %s, err: %v", parts[0], err)
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

func (s *Session) FailTimedOutTasks(now time.Time) {
	if s == nil || s.Tasks == nil {
		return
	}
	for _, task := range s.Tasks.All() {
		if task != nil && !task.Finished() && task.TimedOutAt(now) {
			task.Fail(nil, "task deadline exceeded")
		}
	}
}

func (s *Session) LastCheckinUnix() int64 {
	return s.lastCheckin.Load()
}

func (s *Session) expectedCheckinDeadline() (time.Time, bool) {
	if s == nil {
		return time.Time{}, false
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.expectedCheckinDeadlineLocked()
}

func (s *Session) expectedCheckinDeadlineLocked() (time.Time, bool) {
	lastCheckin := s.LastCheckinUnix()
	if lastCheckin <= 0 {
		return time.Time{}, false
	}
	if strings.TrimSpace(s.Expression) == "" {
		return time.Time{}, false
	}

	parsedExpr, err := cronexpr.Parse(s.Expression)
	if err != nil {
		logs.Log.Debugf("session %s timer expression parse error %q: %v", s.ID, s.Expression, err)
		return time.Time{}, false
	}

	base := time.Unix(lastCheckin, 0)
	nextTime := parsedExpr.Next(base)
	if nextTime.IsZero() || !nextTime.After(base) {
		return time.Time{}, false
	}

	jitter := s.Jitter
	if jitter < 0 {
		jitter = 0
	}

	expectedInterval := nextTime.Sub(base)
	if expectedInterval <= 0 {
		return time.Time{}, false
	}

	multiplier := 1 + jitter
	if multiplier < 10 {
		multiplier = 10
	}

	allowedOffline := time.Duration(float64(expectedInterval)*multiplier) + 10*time.Minute
	return base.Add(allowedOffline), true
}

func (s *Session) isAliveAt(now time.Time) bool {
	if s == nil {
		return false
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.isAliveAtLocked(now)
}

func (s *Session) isAliveAtLocked(now time.Time) bool {
	if s.Type == consts.BindPipeline {
		return true
	}
	deadline, ok := s.expectedCheckinDeadlineLocked()
	if !ok {
		return true
	}
	return !now.After(deadline)
}

func (s *Session) isAlived() bool {
	return s.isAliveAt(time.Now())
}

func (s *Session) ToProtobuf() *clientpb.Session {
	pb := s.ToProtobufLite()
	pb.Tasks = s.Tasks.ToProtobuf()
	return pb
}

func (s *Session) ToProtobufLite() *clientpb.Session {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.toProtobufLiteLocked(time.Now())
}

func (s *Session) toProtobufLiteLocked(now time.Time) *clientpb.Session {
	modules := append([]string(nil), s.Modules...)
	addons := cloneSessionAddons(s.Addons)
	var keyPair *clientpb.KeyPair
	if s.KeyPair != nil {
		keyPair = proto.Clone(s.KeyPair).(*clientpb.KeyPair)
	}

	return &clientpb.Session{
		Type:          s.Type,
		SessionId:     s.ID,
		RawId:         s.RawID,
		Note:          s.Note,
		Name:          s.Name,
		GroupName:     s.Group,
		IsAlive:       s.isAliveAtLocked(now),
		IsPrivilege:   s.IsPrivilege,
		Target:        s.Target,
		PipelineId:    s.PipelineID,
		ListenerId:    s.ListenerID,
		Os:            cloneSysInfoOS(s.Os),
		Process:       cloneSysInfoProcess(s.Process),
		LastCheckin:   s.LastCheckinUnix(),
		Filepath:      s.SessionContext.Filepath,
		Workdir:       s.SessionContext.WorkDir,
		Locate:        s.SessionContext.Locale,
		Proxy:         s.SessionContext.ProxyURL,
		Timer:         &implantpb.Timer{Expression: s.Expression, Jitter: s.Jitter},
		CreatedAt:     s.CreatedAt.Unix(),
		Modules:       modules,
		Addons:        addons,
		KeyPair:       keyPair,
		Data:          s.Marshal(),
		IsInitialized: s.Initialized,
	}
}

func cloneSessionAddons(addons []*implantpb.Addon) []*implantpb.Addon {
	if addons == nil {
		return nil
	}
	cloned := make([]*implantpb.Addon, len(addons))
	for i, addon := range addons {
		if addon != nil {
			cloned[i] = proto.Clone(addon).(*implantpb.Addon)
		}
	}
	return cloned
}

func (s *Session) Save() error {
	return sessionDBSave(s.ToModel())
}

func (s *Session) ToModel() *models.Session {
	s.stateMu.RLock()
	name := s.Name
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
		IsAlive:     s.isAliveAtLocked(time.Now()),
		LastCheckin: s.LastCheckinUnix(),
		DataString:  s.Marshal(),
	}
	s.stateMu.RUnlock()
	artifact, err := sessionDBGetArtifact(name)
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
		Important: true,
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

func (s *Session) SetNote(note string) {
	s.stateMu.Lock()
	s.Note = note
	s.stateMu.Unlock()
}

func (s *Session) SetGroup(group string) {
	s.stateMu.Lock()
	s.Group = group
	s.stateMu.Unlock()
}

func (s *Session) SetTimer(expression string, jitter float64) {
	if s == nil {
		return
	}
	s.stateMu.Lock()
	s.ensureSessionContextLocked()
	s.Expression = expression
	s.Jitter = jitter
	s.stateMu.Unlock()
}

func (s *Session) TimerSnapshot() (string, float64) {
	if s == nil {
		return "", 0
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	if s.SessionContext == nil || s.SessionInfo == nil {
		return "", 0
	}
	return s.Expression, s.Jitter
}

func (s *Session) ensureSessionContextLocked() {
	if s.SessionContext == nil {
		s.SessionContext = &client.SessionContext{}
	}
	if s.SessionInfo == nil {
		s.SessionInfo = &client.SessionInfo{}
	}
}

func (s *Session) RoutingSnapshot() (string, string) {
	_, listenerID, pipelineID := s.ConnectionSnapshot()
	return listenerID, pipelineID
}

func (s *Session) ConnectionSnapshot() (string, string, string) {
	if s == nil {
		return "", "", ""
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.Target, s.ListenerID, s.PipelineID
}

func (s *Session) ReplaceAddons(addons []*implantpb.Addon) {
	if s == nil {
		return
	}
	s.stateMu.Lock()
	s.ensureSessionContextLocked()
	s.Addons = mergeSessionAddons(nil, addons)
	s.stateMu.Unlock()
}

func (s *Session) MergeAddons(addons []*implantpb.Addon) {
	if s == nil || len(addons) == 0 {
		return
	}
	s.stateMu.Lock()
	s.ensureSessionContextLocked()
	s.Addons = mergeSessionAddons(s.Addons, addons)
	s.stateMu.Unlock()
}

func (s *Session) AddonsSnapshot() []*implantpb.Addon {
	if s == nil {
		return nil
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	if s.SessionContext == nil {
		return nil
	}
	return cloneSessionAddons(s.Addons)
}

func (s *Session) HasAddon(name string) bool {
	if s == nil || name == "" {
		return false
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	if s.SessionContext == nil {
		return false
	}
	for _, addon := range s.Addons {
		if addon != nil && addon.GetName() == name {
			return true
		}
	}
	return false
}

func mergeSessionAddons(existing, incoming []*implantpb.Addon) []*implantpb.Addon {
	merged := make([]*implantpb.Addon, 0, len(existing)+len(incoming))
	indexes := make(map[string]int, len(existing)+len(incoming))
	appendOrReplace := func(addon *implantpb.Addon) {
		if addon == nil || addon.GetName() == "" {
			return
		}
		cloned := proto.Clone(addon).(*implantpb.Addon)
		if index, ok := indexes[addon.GetName()]; ok {
			merged[index] = cloned
			return
		}
		indexes[addon.GetName()] = len(merged)
		merged = append(merged, cloned)
	}
	for _, addon := range existing {
		appendOrReplace(addon)
	}
	for _, addon := range incoming {
		appendOrReplace(addon)
	}
	return merged
}

func (s *Session) ApplyModules(modules *implantpb.Modules, appendOnly bool) {
	if modules == nil {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if appendOnly {
		s.Modules = append(s.Modules, modules.GetModules()...)
		if bundleMap := modules.GetBundleMap(); len(bundleMap) > 0 {
			if s.BundleMap == nil {
				s.BundleMap = make(map[string]string)
			}
			for key, value := range bundleMap {
				s.BundleMap[key] = value
			}
		}
		return
	}

	s.Modules = append([]string(nil), modules.GetModules()...)
	bundleMap := modules.GetBundleMap()
	if bundleMap == nil {
		s.BundleMap = nil
		return
	}
	s.BundleMap = make(map[string]string, len(bundleMap))
	for key, value := range bundleMap {
		s.BundleMap[key] = value
	}
}

// GetPipelineEncryptionKey returns the transport encryption key from the
// session's pipeline config. Returns "" if not found.
func (s *Session) GetPipelineEncryptionKey() string {
	pipeline, ok := s.findPipeline()
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
	pipeline, ok := s.findPipeline()
	if ok && pipeline != nil && pipeline.PacketLength > 0 {
		return int(pipeline.PacketLength)
	}
	return config.Int(consts.ConfigMaxPacketLength)
}

func (s *Session) findPipeline() (*clientpb.Pipeline, bool) {
	if s == nil {
		return nil, false
	}
	s.stateMu.RLock()
	pipelineID := s.PipelineID
	listenerID := s.ListenerID
	s.stateMu.RUnlock()
	if pipelineID == "" {
		return nil, false
	}
	if listenerID != "" {
		return Listeners.FindByListener(listenerID, pipelineID)
	}
	return Listeners.Find(pipelineID)
}

func (s *Session) Update(req *clientpb.RegisterSession) {
	if req == nil || req.RegisterData == nil {
		return
	}
	s.stateMu.Lock()
	s.Name = req.RegisterData.Name
	s.PipelineID = req.PipelineId
	s.ListenerID = req.ListenerId
	s.ProxyURL = req.RegisterData.Proxy
	if req.RegisterData.Timer != nil {
		s.Expression = req.RegisterData.Timer.Expression
		s.Jitter = req.RegisterData.Timer.Jitter
	}
	s.Modules = append([]string(nil), req.RegisterData.Module...)
	s.Addons = cloneSessionAddons(req.RegisterData.Addons)
	// SecureManager现在使用固定的100次交互计数，不需要更新间隔

	shouldPublishInit := false
	if req.RegisterData.Sysinfo != nil {
		shouldPublishInit = !s.Initialized
		s.updateSysInfoLocked(req.RegisterData.Sysinfo)
	}
	s.stateMu.Unlock()
	if shouldPublishInit {
		s.Publish(consts.CtrlSessionInit, fmt.Sprintf("session %s init", s.ID), true, true)
	}

	err := s.Save()
	if err != nil {
		logs.Log.Errorf("update session %s info failed in db, %s", s.ID, err.Error())
	}
}

func (s *Session) UpdateSysInfo(info *implantpb.SysInfo) {
	if info == nil {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.updateSysInfoLocked(info)
}

func (s *Session) SetWorkDir(workDir string) {
	s.stateMu.Lock()
	s.WorkDir = workDir
	s.stateMu.Unlock()
}

func (s *Session) updateSysInfoLocked(info *implantpb.SysInfo) {
	s.Initialized = true
	osInfo := mergeSysInfoOS(s.Os, info.GetOs())
	processInfo := mergeSysInfoProcess(s.Process, info.GetProcess())

	osInfo.Name = strings.ToLower(osInfo.Name)
	osInfo.Arch = consts.FormatArch(osInfo.Arch)
	processInfo.Arch = consts.FormatArch(processInfo.Arch)

	s.IsPrivilege = info.IsPrivilege
	if info.Filepath != "" {
		s.Filepath = info.Filepath
	}
	if info.Workdir != "" {
		s.WorkDir = info.Workdir
	}
	s.Os = osInfo
	s.Process = processInfo
}

func mergeSysInfoOS(current *implantpb.Os, incoming *implantpb.Os) *implantpb.Os {
	merged := cloneSysInfoOS(current)
	if isZeroSysInfoOS(merged) {
		merged = &implantpb.Os{}
	}
	if incoming == nil {
		return merged
	}
	if incoming.Name != "" {
		merged.Name = incoming.Name
	}
	if incoming.Version != "" {
		merged.Version = incoming.Version
	}
	if incoming.Release != "" {
		merged.Release = incoming.Release
	}
	if incoming.Arch != "" {
		merged.Arch = incoming.Arch
	}
	if incoming.Username != "" {
		merged.Username = incoming.Username
	}
	if incoming.Hostname != "" {
		merged.Hostname = incoming.Hostname
	}
	if incoming.Locale != "" {
		merged.Locale = incoming.Locale
	}
	if len(incoming.ClrVersion) > 0 {
		merged.ClrVersion = append([]string(nil), incoming.ClrVersion...)
	}
	return merged
}

func mergeSysInfoProcess(current *implantpb.Process, incoming *implantpb.Process) *implantpb.Process {
	merged := cloneSysInfoProcess(current)
	if isZeroSysInfoProcess(merged) {
		merged = &implantpb.Process{}
	}
	if incoming == nil || isZeroSysInfoProcess(incoming) {
		return merged
	}
	if incoming.Name != "" {
		merged.Name = incoming.Name
	}
	if incoming.Pid != 0 {
		merged.Pid = incoming.Pid
	}
	if incoming.Ppid != 0 {
		merged.Ppid = incoming.Ppid
	}
	if incoming.Owner != "" {
		merged.Owner = incoming.Owner
	}
	if incoming.Arch != "" {
		merged.Arch = incoming.Arch
	}
	if incoming.Path != "" {
		merged.Path = incoming.Path
	}
	if incoming.Args != "" {
		merged.Args = incoming.Args
	}
	if incoming.Uid != "" {
		merged.Uid = incoming.Uid
	}
	if hasSignatureMetadata(incoming) {
		merged.Signed = incoming.Signed
		merged.SignatureStatus = incoming.SignatureStatus
		merged.Signer = incoming.Signer
		merged.Issuer = incoming.Issuer
	}
	return merged
}

func cloneSysInfoOS(value *implantpb.Os) *implantpb.Os {
	if value == nil {
		return &implantpb.Os{}
	}
	clone := proto.Clone(value)
	if clone == nil {
		return &implantpb.Os{}
	}
	return clone.(*implantpb.Os)
}

func cloneSysInfoProcess(value *implantpb.Process) *implantpb.Process {
	if value == nil {
		return &implantpb.Process{}
	}
	clone := proto.Clone(value)
	if clone == nil {
		return &implantpb.Process{}
	}
	return clone.(*implantpb.Process)
}

func isZeroSysInfoOS(value *implantpb.Os) bool {
	if value == nil {
		return true
	}
	return value.Name == "" &&
		value.Version == "" &&
		value.Release == "" &&
		value.Arch == "" &&
		value.Username == "" &&
		value.Hostname == "" &&
		value.Locale == "" &&
		len(value.ClrVersion) == 0
}

func isZeroSysInfoProcess(value *implantpb.Process) bool {
	if value == nil {
		return true
	}
	return value.Name == "" &&
		value.Pid == 0 &&
		value.Ppid == 0 &&
		value.Owner == "" &&
		value.Arch == "" &&
		value.Path == "" &&
		value.Args == "" &&
		value.Uid == "" &&
		!value.Signed &&
		value.SignatureStatus == "" &&
		value.Signer == "" &&
		value.Issuer == ""
}

func hasSignatureMetadata(value *implantpb.Process) bool {
	if value == nil {
		return false
	}
	return value.Signed ||
		value.SignatureStatus != "" ||
		value.Signer != "" ||
		value.Issuer != ""
}

func (s *Session) FillSysInfo() {
	s.stateMu.RLock()
	name := s.Name
	s.stateMu.RUnlock()
	artifact, err := sessionDBGetArtifact(name)
	if err != nil {
		logs.Log.Errorf("failed to find atrtifact %s: %s", name, err)
		return
	}
	s.stateMu.Lock()
	s.Os = &implantpb.Os{
		Name: artifact.Os,
		Arch: artifact.Arch,
	}
	s.stateMu.Unlock()
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
	seq := s.Taskseq.Add(1)
	now := time.Now()
	task := &Task{
		Type:      name,
		Total:     total,
		Id:        seq,
		SessionId: s.ID,
		Session:   s,
		DoneCh:    make(chan bool, 1),
		CreatedAt: now,
		Deadline:  now.Add(configs.DefaultTaskTimeout),
	}
	task.Ctx, task.Cancel = context.WithCancel(s.Ctx)
	s.Tasks.Add(task)
	return task
}

// Request
func (s *Session) Request(msg *clientpb.SpiteRequest, stream grpc.ServerStream) error {
	return stream.SendMsg(msg)
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
	val, loaded := s.responses.LoadAndDelete(taskId)
	if loaded {
		close(val.(chan *implantpb.Spite))
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
	if parentSession.Cache != nil {
		parentSession.Cache.Close()
	}
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
	now := time.Now()
	for _, session := range s.All() {
		if session != nil {
			session.FailTimedOutTasks(now)
		}
		if session == nil || session.isAlived() {
			continue
		}
		// Re-check with the latest timestamp to avoid racing with a fresh checkin.
		if session.isAlived() {
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
		// A checkin may have revived the session after it was marked dead but
		// before runtime eviction. Keep it in memory when the latest state is alive.
		if session.isAlived() {
			session.MarkAlive()
			continue
		}
		s.Remove(session.ID)
	}
}

// restoreSecureManagerFromContext rebuilds only the in-memory secure runtime.
// Persisted session hydration must not depend on a live listener or pipeline.
func (s *Session) restoreSecureManagerFromContext() {
	if s == nil || s.SessionContext == nil || s.Secure == nil || !s.Secure.Enable {
		return
	}
	s.SecureManager = NewSecureSpiteManager(s)
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
		logs.Log.Debugf("secure - pipeline_disabled session=%s", s.ID)
		return nil
	}

	if req.RegisterData.Secure == nil || !req.RegisterData.Secure.Enable {
		logs.Log.Debugf("secure - session_secure_disabled session=%s", s.ID)
		return nil
	}

	if s.KeyPair == nil {
		// Populate from pipeline pre-shared keys when available (may be empty
		// in cold-start scenarios where no keys are distributed ahead of time).
		pubKey := ""
		privKey := ""
		if pipeline.Secure.ImplantKeypair != nil {
			pubKey = pipeline.Secure.ImplantKeypair.PublicKey
		}
		if pipeline.Secure.ServerKeypair != nil {
			privKey = pipeline.Secure.ServerKeypair.PrivateKey
		}
		s.KeyPair = &clientpb.KeyPair{
			PublicKey:  pubKey,
			PrivateKey: privKey,
		}
	}

	s.PushCtrl()
	logs.Log.Infof("secure - initialized_session session=%s", s.ID)

	s.SecureManager = NewSecureSpiteManager(s)
	return nil
}

func (s *Session) UpdatePublicKey(key string) {
	s.UpdateKeyPair(key, "")
}

func (s *Session) UpdatePrivateKey(key string) {
	s.UpdateKeyPair("", key)
}

// UpdateKeyPair merges the given non-empty public/private key fields into the
// session's current key pair, syncs the SecureManager, and pushes the update to
// the listener; an empty field leaves the existing value unchanged. The merge
// and store run under a single write lock so concurrent field updates (e.g. a
// re-register updating the public key while a key rotation updates both) cannot
// read the same base snapshot and clobber each other.
func (s *Session) UpdateKeyPair(publicKey string, privateKey string) {
	s.stateMu.Lock()
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
	// next is freshly allocated here and shared with no caller, so no defensive
	// clone is needed before storing it.
	s.SessionContext.KeyPair = next
	if s.SecureManager != nil {
		// Keep the session and manager updates in the same critical section so
		// concurrent callers cannot publish an older manager value after a newer
		// session value has already been stored.
		s.SecureManager.UpdateKeyPair(next)
	}
	s.stateMu.Unlock()
	s.PushCtrl()
}

func (s *Session) PushCtrl() {
	if s == nil {
		return
	}
	snapshot := s.ToProtobufLite()
	lns, err := Listeners.Get(snapshot.ListenerId)
	if err != nil {
		return
	}
	if err := s.Save(); err != nil {
		logs.Log.Errorf("sync session %s persistence failed: %v", s.ID, err)
	}
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl:    consts.CtrlListenerSyncSession,
		Session: snapshot,
	})
}
