package rpc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type testRPCServerStream struct {
	ctx     context.Context
	sendMsg func(interface{}) error
	recvMsg func(interface{}) error
}

func (s *testRPCServerStream) SetHeader(metadata.MD) error  { return nil }
func (s *testRPCServerStream) SendHeader(metadata.MD) error { return nil }
func (s *testRPCServerStream) SetTrailer(metadata.MD)       {}
func (s *testRPCServerStream) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *testRPCServerStream) SendMsg(m interface{}) error {
	if s.sendMsg != nil {
		return s.sendMsg(m)
	}
	return nil
}
func (s *testRPCServerStream) RecvMsg(m interface{}) error {
	if s.recvMsg != nil {
		return s.recvMsg(m)
	}
	return nil
}

var _ grpc.ServerStream = (*testRPCServerStream)(nil)

func TestSleepDispatchesRequestAndUpdatesSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-sleep-session", "rpc-sleep-pipe", true)

	sent := make(chan *clientpb.SpiteRequest, 1)
	pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
		sendMsg: func(m interface{}) error {
			req, ok := m.(*clientpb.SpiteRequest)
			if !ok {
				t.Fatalf("unexpected message type %T", m)
			}
			sent <- req
			return nil
		},
	})
	t.Cleanup(func() { pipelinesCh.Delete(sess.PipelineID) })

	req := &implantpb.Timer{Expression: "*/5 * * * * * *", Jitter: 0.4}
	task, err := (&Server{}).Sleep(incomingSessionContext(sess.ID), req)
	if err != nil {
		t.Fatalf("Sleep failed: %v", err)
	}
	if sess.Expression != req.Expression || sess.Jitter != req.Jitter {
		t.Fatalf("session timer = %q/%v, want %q/%v", sess.Expression, sess.Jitter, req.Expression, req.Jitter)
	}

	select {
	case spiteReq := <-sent:
		if spiteReq.Task == nil || spiteReq.Task.TaskId != task.TaskId {
			t.Fatalf("spite request task = %#v, want task id %d", spiteReq.Task, task.TaskId)
		}
		if spiteReq.Spite == nil || spiteReq.Spite.GetSleepRequest().GetExpression() != req.Expression {
			t.Fatalf("spite request = %#v, want sleep expression %q", spiteReq.Spite, req.Expression)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for spite request")
	}

	deliverTaskResponse(t, sess, task.TaskId, &implantpb.Spite{
		Body: &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
	})
	waitForCondition(t, 2*time.Second, func() bool {
		stored := sess.Tasks.Get(task.TaskId)
		return stored != nil && stored.Finished()
	}, "sleep task to finish")
}

func TestKeepaliveEnablesSessionAfterResponse(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-keepalive-session", "rpc-keepalive-pipe", true)

	pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
		sendMsg: func(m interface{}) error {
			req, ok := m.(*clientpb.SpiteRequest)
			if !ok {
				t.Fatalf("unexpected message type %T", m)
			}
			if req.Spite == nil || req.Spite.Name != consts.ModuleKeepalive {
				t.Fatalf("spite request = %#v, want keepalive request", req.Spite)
			}
			return nil
		},
	})
	t.Cleanup(func() { pipelinesCh.Delete(sess.PipelineID) })

	task, err := (&Server{}).Keepalive(incomingSessionContext(sess.ID), &implantpb.CommonBody{
		BoolArray: []bool{true},
	})
	if err != nil {
		t.Fatalf("Keepalive failed: %v", err)
	}
	if sess.IsKeepaliveEnabled() {
		t.Fatal("keepalive should only toggle after response callback")
	}

	deliverTaskResponse(t, sess, task.TaskId, &implantpb.Spite{
		Name: consts.ModuleKeepalive,
		Body: &implantpb.Spite_Common{Common: &implantpb.CommonBody{Name: consts.ModuleKeepalive}},
	})
	waitForCondition(t, 2*time.Second, sess.IsKeepaliveEnabled, "keepalive to become enabled")
}

func TestInfoUpdatesSessionSysinfoFromResponse(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-info-session", "rpc-info-pipe", true)

	pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
		sendMsg: func(interface{}) error { return nil },
	})
	t.Cleanup(func() { pipelinesCh.Delete(sess.PipelineID) })

	task, err := (&Server{}).Info(incomingSessionContext(sess.ID), &implantpb.Request{Name: consts.ModuleSysInfo})
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	deliverTaskResponse(t, sess, task.TaskId, &implantpb.Spite{
		Body: &implantpb.Spite_Sysinfo{Sysinfo: &implantpb.SysInfo{
			Os: &implantpb.Os{
				Name: "LINUX",
				Arch: "x86_64",
			},
			Process: &implantpb.Process{Name: "agent.bin"},
		}},
	})
	waitForCondition(t, 2*time.Second, func() bool {
		return sess.Os != nil && sess.Os.Name == "linux" && sess.Os.Arch == "x64"
	}, "session sysinfo update")
}

func TestGetSessionReturnsDatabaseRecordWithoutRecovery(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-dbonly-session", "rpc-dbonly-pipe", false)

	got, err := (&Server{}).GetSession(context.Background(), &clientpb.SessionRequest{SessionId: sess.ID})
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if got == nil || got.SessionId != sess.ID {
		t.Fatalf("GetSession = %#v, want session %s", got, sess.ID)
	}
	if _, err := core.Sessions.Get(sess.ID); err == nil {
		t.Fatal("GetSession should not recover a db-only session into memory")
	}
}

func TestSessionManageUpdatesActiveAndDatabaseOnlySessions(t *testing.T) {
	env := newRPCTestEnv(t)
	active := env.seedSession(t, "rpc-manage-active", "rpc-manage-pipe", true)
	inactive := env.seedSession(t, "rpc-manage-inactive", "rpc-manage-pipe", false)

	if _, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: active.ID,
		Op:        "note",
		Arg:       "blue-team",
	}); err != nil {
		t.Fatalf("SessionManage note failed: %v", err)
	}
	if active.Note != "blue-team" {
		t.Fatalf("active note = %q, want blue-team", active.Note)
	}
	activeModel, err := env.getSession(active.ID)
	if err != nil {
		t.Fatalf("GetSession(active) failed: %v", err)
	}
	if activeModel.Note != "blue-team" {
		t.Fatalf("active model note = %q, want blue-team", activeModel.Note)
	}

	if _, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: inactive.ID,
		Op:        "group",
		Arg:       "operators",
	}); err != nil {
		t.Fatalf("SessionManage group failed: %v", err)
	}
	inactiveModel, err := env.getSession(inactive.ID)
	if err != nil {
		t.Fatalf("GetSession(inactive) failed: %v", err)
	}
	if inactiveModel.GroupName != "operators" {
		t.Fatalf("inactive group = %q, want operators", inactiveModel.GroupName)
	}

	if _, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: inactive.ID,
		Op:        "delete",
	}); err != nil {
		t.Fatalf("SessionManage delete failed: %v", err)
	}
	model, err := db.FindSession(inactive.ID)
	if err != nil {
		t.Fatalf("FindSession after delete failed: %v", err)
	}
	if model != nil {
		t.Fatalf("expected deleted session lookup to return nil, got %#v", model)
	}
}

func TestCheckinRecoversRemovedSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-checkin-session", "rpc-checkin-pipe", false)

	if err := db.RemoveSession(sess.ID); err != nil {
		t.Fatalf("RemoveSession failed: %v", err)
	}

	if _, err := (&Server{}).Checkin(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 1}); err != nil {
		t.Fatalf("Checkin failed: %v", err)
	}

	recovered, err := core.Sessions.Get(sess.ID)
	if err != nil {
		t.Fatalf("expected recovered session in memory: %v", err)
	}
	if recovered.LastCheckinUnix() == 0 {
		t.Fatal("expected recovered session last checkin to be updated")
	}
	model, err := env.getSession(sess.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if model == nil || model.SessionId != sess.ID {
		t.Fatalf("recovered model = %#v, want session %s", model, sess.ID)
	}
}

func TestRegisterRejectsNilOrIncompleteRequests(t *testing.T) {
	if _, err := (&Server{}).Register(context.Background(), nil); !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("Register(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}

	if _, err := (&Server{}).Register(context.Background(), &clientpb.RegisterSession{}); !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("Register(empty) error = %v, want %v", err, types.ErrMissingRequestField)
	}

	if _, err := (&Server{}).Register(context.Background(), &clientpb.RegisterSession{
		SessionId:    "",
		RegisterData: &implantpb.Register{Name: "agent-a"},
	}); !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("Register(empty session id) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
}

func TestCheckinRejectsNilPing(t *testing.T) {
	if _, err := (&Server{}).Checkin(incomingSessionContext("rpc-checkin-nil"), nil); !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("Checkin(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func incomingSessionContext(sessionID string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"session_id", sessionID,
		"callee", consts.CalleeCMD,
	))
}

func deliverTaskResponse(t testing.TB, sess *core.Session, taskID uint32, spite *implantpb.Spite) {
	t.Helper()

	respCh, ok := sess.GetResp(taskID)
	if !ok {
		t.Fatalf("response channel for task %d not found", taskID)
	}
	if err := deliverSpiteResponse(respCh, spite); err != nil {
		t.Fatalf("deliverSpiteResponse failed: %v", err)
	}
}

type rpcTestEnv struct{}

func newRPCTestEnv(t testing.TB) *rpcTestEnv {
	t.Helper()

	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	if err := os.MkdirAll(configs.ServerRootPath, 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	oldDBClient := db.Client
	t.Cleanup(func() {
		db.Client = oldDBClient
	})
	var dbErr error
	db.Client, dbErr = db.NewDBClient(nil)
	if dbErr != nil {
		t.Fatalf("NewDBClient failed: %v", dbErr)
	}

	oldTicker := core.GlobalTicker
	core.GlobalTicker = core.NewTicker()
	t.Cleanup(func() {
		core.GlobalTicker.RemoveAll()
		core.GlobalTicker = oldTicker
	})

	oldBroker := core.EventBroker
	oldSessions := core.Sessions
	oldListenerMap := core.Listeners.Map
	oldJobsMap := core.Jobs.Map
	t.Cleanup(func() {
		if core.EventBroker != nil {
			core.EventBroker.Stop()
		}
		core.EventBroker = oldBroker
		core.Sessions = oldSessions
		core.Listeners.Map = oldListenerMap
		core.Jobs.Map = oldJobsMap
	})

	core.Listeners.Map = &sync.Map{}
	core.Jobs.Map = &sync.Map{}
	broker := core.NewBroker()
	waitEventBrokerReady(t, broker)
	core.NewSessions()

	return &rpcTestEnv{}
}

func (e *rpcTestEnv) seedSession(t testing.TB, sessionID, pipelineName string, active bool) *core.Session {
	t.Helper()

	listener, err := core.Listeners.Get("test-listener")
	if err != nil {
		listener = core.NewListener("test-listener", "127.0.0.1")
		core.Listeners.Add(listener)
	}
	listener.AddPipeline(&clientpb.Pipeline{
		Name:       pipelineName,
		ListenerId: listener.Name,
		Ip:         "127.0.0.1",
		Type:       consts.TCPPipeline,
		Secure:     &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       pipelineName,
				ListenerId: listener.Name,
				Host:       "127.0.0.1",
				Port:       4444,
			},
		},
	})

	req := &clientpb.RegisterSession{
		Type:       consts.TCPPipeline,
		SessionId:  sessionID,
		RawId:      1,
		PipelineId: pipelineName,
		ListenerId: "test-listener",
		Target:     "127.0.0.1",
		RegisterData: &implantpb.Register{
			Name: "seed-artifact",
			Timer: &implantpb.Timer{
				Expression: "* * * * *",
			},
			Sysinfo: &implantpb.SysInfo{
				Os: &implantpb.Os{
					Name: "windows",
					Arch: "amd64",
				},
				Process: &implantpb.Process{
					Name: "seed.exe",
				},
			},
		},
	}

	sess, err := core.RegisterSession(req)
	if err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}
	sess.SetLastCheckin(time.Now().Unix())
	if err := sess.Save(); err != nil {
		t.Fatalf("session.Save failed: %v", err)
	}
	if active {
		core.Sessions.Add(sess)
	}
	return sess
}

func (e *rpcTestEnv) getSession(sessionID string) (*clientpb.Session, error) {
	model, err := db.FindSession(sessionID)
	if err != nil || model == nil {
		return nil, err
	}
	return model.ToProtobuf(), nil
}

func waitForCondition(t testing.TB, timeout time.Duration, cond func() bool, description string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}
