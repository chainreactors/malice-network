package rpc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *Server) Register(ctx context.Context, req *clientpb.RegisterSession) (*clientpb.Empty, error) {
	if req == nil || req.RegisterData == nil {
		return nil, types.ErrMissingRequestField
	}
	if req.SessionId == "" {
		return nil, types.ErrInvalidSessionID
	}
	var err error
	isNewSession := false
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		isNewSession = true
		sess, err = core.RegisterSession(req)
		if err != nil {
			return nil, err
		}
		sess.SetLastCheckin(getTimestamp(ctx))
		err = db.CreateOrRecoverSession(sess.ToModel())
		if err != nil {
			return nil, err
		}
		sess.Publish(consts.CtrlSessionRegister, fmt.Sprintf("new session %s from %s start at %s", sess.Abstract(), sess.Target, sess.PipelineID), true, true)
		logs.Log.Importantf("new session %s from %s", sess.ID, sess.PipelineID)
		core.Sessions.Add(sess)
	} else {
		logs.Log.Infof("session %s re-register", sess.ID)
		sess.SetLastCheckin(getTimestamp(ctx))
		sess.Update(req)
		sess.Publish(consts.CtrlSessionUpdate, fmt.Sprintf("%s from %s re-registered at %s", sess.Abstract(), sess.Target, sess.PipelineID), true, true)
		core.Sessions.Add(sess)
	}

	if sess.SecureManager != nil && req.RegisterData != nil && req.RegisterData.Secure != nil {
		publicKey := req.RegisterData.Secure.PublicKey
		if publicKey != "" {
			sess.UpdatePublicKey(publicKey)
		}
	}

	// 安全模式下仅在首次注册时触发密钥交换，避免每次重连都触发轮换。
	if isNewSession && sess.SecureManager != nil {
		err := rpc.triggerKeyExchange(ctx, sess)
		if err != nil {
			return nil, err
		}
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) SysInfo(ctx context.Context, req *implantpb.SysInfo) (*clientpb.Empty, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	id, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := core.Sessions.Get(id)
	if err != nil {
		return nil, err
	}
	sess.UpdateSysInfo(req)
	sess.SaveAndNotify("")
	return &clientpb.Empty{}, nil
}

func (rpc *Server) Checkin(ctx context.Context, req *implantpb.Ping) (*clientpb.Empty, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	sid, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	var sess *core.Session
	reborn := false
	if sess, err = core.Sessions.Get(sid); err != nil {
		dbSess, err := db.FindSession(sid)
		if err != nil {
			return nil, err
		}
		if dbSess == nil {
			// session was soft-deleted, try to recover it
			dbSess, err = db.RecoverRemovedSession(sid)
			if err != nil || dbSess == nil {
				return nil, err
			}
		}
		dbSess.LastCheckin = getTimestamp(ctx)
		sess, err = core.RecoverSession(dbSess)
		if err != nil {
			return nil, err
		}
		core.Sessions.Add(sess)
		reborn = true
		logs.Log.Debugf("recover session %s", sid)
	} else if sess.MarkAlive() {
		reborn = true
	}
	sess.SetLastCheckin(getTimestamp(ctx))
	if err := sess.Save(); err != nil {
		logs.Log.Errorf("save session %s checkin failed: %s", sess.ID, err.Error())
	}
	if reborn {
		sess.Publish(consts.CtrlSessionReborn, fmt.Sprintf("session %s from %s reborn at %s", sess.Abstract(), sess.Target, sess.PipelineID), true, true)
	}
	sess.Publish(consts.CtrlSessionCheckin, "", false, false)

	// 增加密钥轮换计数器并检查是否需要轮换
	if sess.SecureManager != nil {
		if sess.SecureManager.ShouldRotateKey() {
			err = rpc.triggerKeyExchange(ctx, sess)
			if err != nil {
				return nil, err
			}
		} else {
			sess.SecureManager.IncrementCounter()
		}
	}

	return &clientpb.Empty{}, nil
}

// sleep
func (rpc *Server) Sleep(ctx context.Context, req *implantpb.Timer) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgEmpty)
	if session, err := getSession(ctx); err == nil {
		session.Jitter = req.Jitter
		session.Expression = req.Expression
		if err := session.SaveAndNotify(""); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	return greq.Task.ToProtobuf(), nil
}

// keepalive - enable/disable duplex mode
func (rpc *Server) Keepalive(ctx context.Context, req *implantpb.CommonBody) (*clientpb.Task, error) {
	req.Name = consts.ModuleKeepalive
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgKeepalive, func(spite *implantpb.Spite) {
		if session, err := getSession(ctx); err == nil {
			enable := len(req.BoolArray) > 0 && req.BoolArray[0]
			session.SetKeepalive(enable)
		}
	})
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Suicide(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleSuicide, types.MsgEmpty)
}

func (rpc *Server) Switch(ctx context.Context, req *implantpb.Switch) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) InitBindSession(ctx context.Context, req *implantpb.Init) (*clientpb.Empty, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	_, err = rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func hasIntersection(slice1, slice2 []uint32) bool {
	set := make(map[uint32]struct{})

	for _, v := range slice1 {
		set[v] = struct{}{}
	}

	for _, v := range slice2 {
		if _, exists := set[v]; exists {
			return true
		}
	}

	return false
}

var pollingRuntimes sync.Map

type pollingRuntime struct {
	mu         sync.RWMutex
	id         string
	sessionID  string
	interval   uint64
	tasks      []uint32
	force      bool
	running    bool
	startedAt  int64
	lastTickAt int64
	lastError  string
	cancel     context.CancelFunc
	done       chan struct{}
}

func pollingRuntimeKey(sessionID string) string {
	return "polling:" + sessionID
}

func pollingRuntimeID(sessionID string) string {
	return "bind-polling:" + sessionID
}

func pollingInterval(interval uint64) uint64 {
	if interval == 0 {
		return uint64(time.Second)
	}
	return interval
}

func validatePollingRequest(req *clientpb.Polling) error {
	if req == nil {
		return types.ErrMissingSessionRequestField
	}
	if req.SessionId == "" {
		return types.ErrInvalidSessionID
	}
	return nil
}

func getOrCreatePollingRuntime(req *clientpb.Polling) (*pollingRuntime, bool) {
	key := pollingRuntimeKey(req.SessionId)
	rt := &pollingRuntime{sessionID: req.SessionId}
	actual, loaded := pollingRuntimes.LoadOrStore(key, rt)
	return actual.(*pollingRuntime), loaded
}

func pollingState(rt *pollingRuntime) *clientpb.PollingState {
	if rt == nil {
		return &clientpb.PollingState{}
	}
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return &clientpb.PollingState{
		Id:         rt.id,
		SessionId:  rt.sessionID,
		Running:    rt.running,
		Interval:   rt.interval,
		Tasks:      append([]uint32(nil), rt.tasks...),
		Force:      rt.force,
		StartedAt:  rt.startedAt,
		LastTickAt: rt.lastTickAt,
		LastError:  rt.lastError,
	}
}

func bindPollingRunning(sessionID string) bool {
	val, ok := pollingRuntimes.Load(pollingRuntimeKey(sessionID))
	if !ok || val == nil {
		return false
	}
	rt, ok := val.(*pollingRuntime)
	if !ok {
		return false
	}
	return pollingState(rt).Running
}

func sendBindPing(sess *core.Session) error {
	streamVal, ok := pipelinesCh.Load(sess.PipelineID)
	if !ok || streamVal == nil {
		return fmt.Errorf("bind pipeline %s unavailable for session %s", sess.PipelineID, sess.ID)
	}
	return sess.Request(
		&clientpb.SpiteRequest{Session: sess.ToProtobufLite(), Task: nil, Spite: types.BuildPingSpite()},
		streamVal.(grpc.ServerStream))
}

func (rt *pollingRuntime) start(req *clientpb.Polling) (context.Context, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.running {
		return nil, status.Errorf(codes.AlreadyExists, "polling already running for session %s", req.SessionId)
	}
	id := req.Id
	if id == "" {
		id = pollingRuntimeID(req.SessionId)
	}
	ctx, cancel := context.WithCancel(context.Background())
	rt.id = id
	rt.sessionID = req.SessionId
	rt.interval = pollingInterval(req.Interval)
	rt.tasks = append([]uint32(nil), req.Tasks...)
	rt.force = req.Force
	rt.running = true
	rt.startedAt = time.Now().Unix()
	rt.lastTickAt = 0
	rt.lastError = ""
	rt.cancel = cancel
	rt.done = make(chan struct{})
	return ctx, nil
}

func (rt *pollingRuntime) stopAndWait(timeout time.Duration) {
	rt.mu.RLock()
	cancel := rt.cancel
	done := rt.done
	rt.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func (rt *pollingRuntime) markTick() {
	rt.mu.Lock()
	rt.lastTickAt = time.Now().Unix()
	rt.mu.Unlock()
}

func (rt *pollingRuntime) markError(err error) {
	if err == nil {
		return
	}
	rt.mu.Lock()
	rt.lastError = err.Error()
	rt.mu.Unlock()
}

func (rt *pollingRuntime) finish() {
	rt.mu.Lock()
	if rt.running {
		rt.running = false
	}
	done := rt.done
	rt.cancel = nil
	rt.done = nil
	rt.mu.Unlock()
	if done != nil {
		close(done)
	}
}

func (rpc *Server) Polling(ctx context.Context, req *clientpb.Polling) (*clientpb.Empty, error) {
	if err := validatePollingRequest(req); err != nil {
		return nil, err
	}
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, types.ErrNotFoundSession
	}
	rt, _ := getOrCreatePollingRuntime(req)
	pollCtx, err := rt.start(req)
	if err != nil {
		return nil, err
	}

	state := pollingState(rt)
	label := fmt.Sprintf("polling:%s:%s", sess.ID, state.Id)
	core.GoGuarded(label, func() error {
		defer func() {
			rt.finish()
			logs.Log.Debugf("polling:%s %s done", state.Id, sess.ID)
		}()
		logs.Log.Debugf("polling:%s %s, interval %d", state.Id, sess.ID, state.Interval)
		for {
			select {
			case <-pollCtx.Done():
				return nil
			default:
			}
			if !state.Force {
				// 如果不为force, 且所有需要等待的任务都已经完成, 则退出轮询
				tasks := sess.Tasks.All()
				var notfinishedId []uint32
				for _, task := range tasks {
					if task.Finished() {
						continue
					}
					notfinishedId = append(notfinishedId, task.Id)
				}

				if !hasIntersection(state.Tasks, notfinishedId) {
					break
				}
			}
			if err := sendBindPing(sess); err != nil {
				err = fmt.Errorf("polling request failed for session %s: %w", sess.ID, err)
				rt.markError(err)
				return err
			}
			rt.markTick()
			select {
			case <-pollCtx.Done():
				return nil
			case <-time.After(time.Duration(state.Interval)):
			}
		}
		return nil
	}, core.CombineErrorHandlers(
		core.LogGuardedError(label),
		func(err error) {
			if core.EventBroker == nil {
				return
			}
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventSession,
				Op:        consts.CtrlSessionError,
				Session:   sess.ToProtobufLite(),
				Message:   fmt.Sprintf("polling %s failed", state.Id),
				Err:       core.ErrorText(err),
				Important: true,
			})
		},
	))
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopPolling(ctx context.Context, req *clientpb.Polling) (*clientpb.Empty, error) {
	if err := validatePollingRequest(req); err != nil {
		return nil, err
	}
	val, ok := pollingRuntimes.Load(pollingRuntimeKey(req.SessionId))
	if !ok {
		return &clientpb.Empty{}, nil
	}
	val.(*pollingRuntime).stopAndWait(2 * time.Second)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) PollingStatus(ctx context.Context, req *clientpb.Polling) (*clientpb.PollingState, error) {
	if err := validatePollingRequest(req); err != nil {
		return nil, err
	}
	val, ok := pollingRuntimes.Load(pollingRuntimeKey(req.SessionId))
	if !ok {
		return &clientpb.PollingState{
			Id:        pollingRuntimeID(req.SessionId),
			SessionId: req.SessionId,
		}, nil
	}
	return pollingState(val.(*pollingRuntime)), nil
}

// triggerKeyExchange 自动触发密钥交换流程
func (rpc *Server) triggerKeyExchange(ctx context.Context, sess *core.Session) error {
	// 构建密钥交换请求
	keyPair, err := cryptography.RandomAgeKeyPair()
	if err != nil {
		return err
	}

	timestamp := uint64(time.Now().Unix())
	nonce := cryptography.RandomString(16)

	// 计算 HMAC-SHA256 签名: HMAC(transport_key, public_key || timestamp || nonce)
	var signature []byte
	if transportKey := sess.GetPipelineEncryptionKey(); transportKey != "" {
		mac := hmac.New(sha256.New, []byte(transportKey))
		mac.Write([]byte(keyPair.Public))
		mac.Write([]byte(fmt.Sprintf("%d", timestamp)))
		mac.Write([]byte(nonce))
		signature = mac.Sum(nil)
	}

	// 创建请求
	req := &implantpb.KeyExchangeRequest{
		PublicKey: keyPair.Public,
		Timestamp: timestamp,
		Nonce:     nonce,
		Signature: signature,
	}
	// Reset counter only on successful exchange — if the callback fires, the
	// implant accepted the request and sent back its new public key. Resetting
	// before the response arrives would swallow a failed attempt and delay the
	// next retry by a full rotation budget (100 checkins).
	_, err = rpc.GenericInternal(ctx, req, consts.ModuleKeyExchange, func(spite *implantpb.Spite) {
		resp := spite.GetKeyExchangeResponse()
		if resp == nil {
			return
		}
		sess.SecureManager.ResetCounters()
		sess.UpdateKeyPairFieldsAndPushCtrl(resp.PublicKey, keyPair.Private)
	})
	return err
}
