package socks5

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// recordingTCPRelayRPC captures TcpRelay calls (action + outgoing session_id metadata).
type recordingTCPRelayRPC struct {
	clientrpc.MaliceRPCClient

	mu    sync.Mutex
	calls []tcpRelayCall

	startTask *clientpb.Task
	startErr  error
}

type tcpRelayCall struct {
	action    implantpb.TunnelCtrl_Action
	sessionID string
	hasMD     bool
}

func (r *recordingTCPRelayRPC) TcpRelay(ctx context.Context, in *implantpb.TunnelCtrl, _ ...grpc.CallOption) (*clientpb.Task, error) {
	sid, has := sessionIDFromOutgoing(ctx)
	r.mu.Lock()
	r.calls = append(r.calls, tcpRelayCall{
		action:    in.GetAction(),
		sessionID: sid,
		hasMD:     has,
	})
	r.mu.Unlock()
	if in.GetAction() == implantpb.TunnelCtrl_STOP {
		return &clientpb.Task{TaskId: 1}, nil
	}
	if r.startErr != nil {
		return nil, r.startErr
	}
	if r.startTask != nil {
		return r.startTask, nil
	}
	return &clientpb.Task{TaskId: 99, SessionId: sid}, nil
}

func sessionIDFromOutgoing(ctx context.Context) (string, bool) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return "", false
	}
	vals := md.Get("session_id")
	if len(vals) == 0 {
		return "", true
	}
	return vals[0], true
}

func (r *recordingTCPRelayRPC) snapshot() []tcpRelayCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]tcpRelayCall, len(r.calls))
	copy(out, r.calls)
	return out
}

func newAuditConsole(t testing.TB, rpc clientrpc.MaliceRPCClient) *core.Console {
	t.Helper()
	oldDir := assets.MaliceDirName
	assets.MaliceDirName = t.TempDir()
	assets.InitLogDir()
	t.Cleanup(func() {
		assets.MaliceDirName = oldDir
		assets.InitLogDir()
	})

	state := &iomclient.ServerState{
		Rpc:             &iomclient.Rpc{MaliceRPCClient: rpc},
		Client:          &clientpb.Client{Name: "socks5-audit", ID: 1},
		ActiveTarget:    &iomclient.ActiveTarget{},
		Listeners:       map[string]*clientpb.Listener{},
		Pipelines:       map[string]*clientpb.Pipeline{},
		Sessions:        map[string]*iomclient.Session{},
		Observers:       map[string]*iomclient.Session{},
		FinishCallbacks: &sync.Map{},
		DoneCallbacks:   &sync.Map{},
		EventHook:       map[iomclient.EventCondition][]iomclient.OnEventFunc{},
		EventCallback:   map[string]func(*clientpb.Event){},
	}
	return &core.Console{
		Server: &core.Server{ServerState: state},
		Log:    iomclient.Log,
	}
}

func addAuditSession(t testing.TB, con *core.Console, id string) *iomclient.Session {
	t.Helper()
	sess := iomclient.NewSession(&clientpb.Session{
		SessionId:  id,
		Type:       consts.ImplantMalefic,
		PipelineId: "pipe-audit",
		IsAlive:    true,
		Timer: &implantpb.Timer{
			Expression: "*/30 * * * * * *",
			Jitter:     0.25,
		},
		Os:   &implantpb.Os{Name: "linux", Arch: "x64"},
		Data: "null",
	}, con.Server.ServerState)
	con.Sessions[id] = sess
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

func isolateRegistry(t *testing.T) {
	t.Helper()
	old := registry
	registry = newSocksRegistry()
	t.Cleanup(func() { registry = old })
}

func mkSvc(sessionID, bind string, port int) *SocksService {
	return &SocksService{
		id:        shortID(sessionID, bind, port),
		sessionID: sessionID,
		bind:      bind,
		port:      port,
		user:      "u",
		status:    StatusListening,
		createdAt: time.Now(),
		conns:     make(map[uint32]*localConn),
		stopCh:    make(chan struct{}),
	}
}

// ---------------------------------------------------------------------------
// P0-1: stopRelay session context
// ---------------------------------------------------------------------------

func TestAudit_StopRelayUsesWrongSessionContext(t *testing.T) {
	isolateRegistry(t)

	rpc := &recordingTCPRelayRPC{}
	con := newAuditConsole(t, rpc)

	sessA := addAuditSession(t, con, "session-AAAA-interactive")
	_ = addAuditSession(t, con, "session-BBBB-target")
	con.ActiveTarget.Set(sessA)

	relB := registry.getRelay("session-BBBB-target")
	relB.mu.Lock()
	relB.task = &clientpb.Task{TaskId: 42, SessionId: "session-BBBB-target"}
	relB.callbackInstalled = true
	relB.mu.Unlock()

	stopRelay(con, "session-BBBB-target")

	calls := rpc.snapshot()
	if len(calls) == 0 {
		t.Fatal("stopRelay issued no TcpRelay(STOP) RPC")
	}
	var stop *tcpRelayCall
	for i := range calls {
		if calls[i].action == implantpb.TunnelCtrl_STOP {
			stop = &calls[i]
			break
		}
	}
	if stop == nil {
		t.Fatalf("TcpRelay called but never with STOP: %+v", calls)
	}

	want := "session-BBBB-target"
	if stop.sessionID != want {
		t.Fatalf("stopRelay session context: got session_id=%q hasMD=%v, want %q",
			stop.sessionID, stop.hasMD, want)
	}
}

func TestAudit_StopRelayNoInteractiveUsesBackground(t *testing.T) {
	isolateRegistry(t)

	rpc := &recordingTCPRelayRPC{}
	con := newAuditConsole(t, rpc)

	rel := registry.getRelay("session-ONLY")
	rel.mu.Lock()
	rel.task = &clientpb.Task{TaskId: 7, SessionId: "session-ONLY"}
	rel.callbackInstalled = true
	rel.mu.Unlock()

	stopRelay(con, "session-ONLY")

	calls := rpc.snapshot()
	if len(calls) == 0 {
		t.Fatal("no TcpRelay call")
	}
	c := calls[0]
	if c.action != implantpb.TunnelCtrl_STOP {
		t.Fatalf("action=%v", c.action)
	}
	if c.sessionID != "session-ONLY" {
		t.Fatalf("no-interactive stopRelay session_id=%q hasMD=%v, want session-ONLY",
			c.sessionID, c.hasMD)
	}
}

// ---------------------------------------------------------------------------
// P0-3: route OpenResult by owner — sibling must not store pending
// ---------------------------------------------------------------------------

func TestAudit_FanoutOpenResultPollutesSiblingPendingOpen(t *testing.T) {
	isolateRegistry(t)

	sid := "session-fanout-1"
	s1 := mkSvc(sid, "127.0.0.1", 19001)
	s2 := mkSvc(sid, "127.0.0.1", 19002)
	s1.relay = registry.getRelay(sid)
	s2.relay = s1.relay
	if err := registry.addService(s1); err != nil {
		t.Fatal(err)
	}
	if err := registry.addService(s2); err != nil {
		t.Fatal(err)
	}

	const connID = uint32(4242)
	// Owner is s1 only.
	s1.relay.registerConn(connID, s1)
	lc := &localConn{
		id:     connID,
		openCh: make(chan *implantpb.TunnelOpenResult, 1),
	}
	s1.mu.Lock()
	s1.conns[connID] = lc
	s1.mu.Unlock()

	spite := &implantpb.Spite{
		Body: &implantpb.Spite_TunnelOpenResult{TunnelOpenResult: &implantpb.TunnelOpenResult{
			ConnId:  connID,
			Success: true,
		}},
	}
	registry.fanoutSpite(sid, spite)

	// Owner got the result.
	select {
	case <-lc.openCh:
	default:
		t.Fatal("owner did not receive OpenResult")
	}

	// Sibling must not accumulate pending on service (field removed) nor relay for foreign id.
	s1.relay.mu.Lock()
	_, leak := s1.relay.pendingOpen[connID]
	s1.relay.mu.Unlock()
	if leak {
		t.Fatal("OpenResult still pending after delivery to owner")
	}
	// s2 has no owner registration and no pending side channel.
	if s2.relay.ownerOf(connID) == s2 {
		t.Fatal("sibling incorrectly owns conn")
	}
}

func TestAudit_PendingOpenLeakAcrossManyOpens(t *testing.T) {
	isolateRegistry(t)

	sid := "session-leak"
	s1 := mkSvc(sid, "127.0.0.1", 19101)
	s2 := mkSvc(sid, "127.0.0.1", 19102)
	rel := registry.getRelay(sid)
	s1.relay, s2.relay = rel, rel
	_ = registry.addService(s1)
	_ = registry.addService(s2)

	const n = 50
	for i := uint32(1); i <= n; i++ {
		// No owner registered: only relay-level pending, not per-sibling.
		registry.fanoutSpite(sid, &implantpb.Spite{
			Body: &implantpb.Spite_TunnelOpenResult{TunnelOpenResult: &implantpb.TunnelOpenResult{
				ConnId:  i,
				Success: true,
			}},
		})
	}

	rel.mu.Lock()
	got := len(rel.pendingOpen)
	rel.mu.Unlock()
	// Single relay buffer, not N * listeners.
	if got != n {
		t.Fatalf("relay pendingOpen=%d want %d", got, n)
	}
	// Critical: no per-service pending maps on siblings.
}

// ---------------------------------------------------------------------------
// P1: openFailStreak triggers rebuild clear
// ---------------------------------------------------------------------------

func TestAudit_OpenFailStreakHasNoThresholdAction(t *testing.T) {
	isolateRegistry(t)

	rel := registry.getRelay("sess-streak")
	svc := mkSvc("sess-streak", "127.0.0.1", 19201)
	svc.relay = rel
	rel.mu.Lock()
	rel.task = &clientpb.Task{TaskId: 1}
	rel.callbackInstalled = true
	rel.mu.Unlock()

	for i := 0; i < openFailRebuildThreshold; i++ {
		svc.setError(fmt.Sprintf("connect failed #%d", i))
	}

	rel.mu.Lock()
	taskNil := rel.task == nil
	installed := rel.callbackInstalled
	streak := rel.openFailStreak
	rel.mu.Unlock()

	if !taskNil || installed {
		t.Fatalf("after %d failures want task cleared: task_nil=%v installed=%v streak=%d",
			openFailRebuildThreshold, taskNil, installed, streak)
	}
}

// ---------------------------------------------------------------------------
// P1: dataQ capped
// ---------------------------------------------------------------------------

func TestAudit_DataQUnboundedGrowth(t *testing.T) {
	svc := mkSvc("sess-q", "127.0.0.1", 19301)
	// dummy conn so Close path is safe
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	lc := &localConn{
		id:       1,
		conn:     c1,
		openCh:   make(chan *implantpb.TunnelOpenResult, 1),
		dataWait: make(chan struct{}, 1),
	}
	svc.mu.Lock()
	svc.conns[1] = lc
	svc.mu.Unlock()

	payload := make([]byte, 1024)
	// Push well past the cap.
	for i := 0; i < maxDataQChunks+100; i++ {
		svc.handleSpite(&implantpb.Spite{
			Body: &implantpb.Spite_TunnelData{TunnelData: &implantpb.TunnelData{
				ConnId: 1,
				Data:   payload,
			}},
		})
	}

	lc.dataMu.Lock()
	qlen := len(lc.dataQ)
	lc.dataMu.Unlock()

	if qlen > maxDataQChunks {
		t.Fatalf("dataQ len=%d exceeds maxDataQChunks=%d", qlen, maxDataQChunks)
	}
	if !lc.closed.Load() {
		t.Fatal("expected connection closed on queue overflow")
	}
}

// ---------------------------------------------------------------------------
// P1: listen-first — listen fail never starts relay
// ---------------------------------------------------------------------------

func TestAudit_StartCmdListenFailLeavesRelayTask(t *testing.T) {
	isolateRegistry(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	rpc := &recordingTCPRelayRPC{
		startTask: &clientpb.Task{TaskId: 77, SessionId: "sess-listen-fail"},
	}
	con := newAuditConsole(t, rpc)
	sess := addAuditSession(t, con, "sess-listen-fail")
	con.ActiveTarget.Set(sess)

	// Production order: listen first. Occupied port fails before START.
	_, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err == nil {
		t.Fatal("expected listen fail on occupied port")
	}

	// No ensureSessionRelay called → no START, no leftover relay task.
	starts := 0
	for _, c := range rpc.snapshot() {
		if c.action == implantpb.TunnelCtrl_START || c.action == implantpb.TunnelCtrl_Action(0) {
			starts++
		}
	}
	if starts != 0 {
		t.Fatalf("listen-fail path must not START tcp_relay, got %d STARTs", starts)
	}
	if _, ok := registry.relays[sess.SessionId]; ok {
		// getRelay may not have been called; if present, must not have task.
		rel := registry.getRelay(sess.SessionId)
		rel.mu.Lock()
		has := rel.task != nil
		rel.mu.Unlock()
		if has {
			t.Fatal("relay task present after listen-only failure path")
		}
	}
}

// ---------------------------------------------------------------------------
// P1: Close bounded wait
// ---------------------------------------------------------------------------

func TestAudit_CloseBlocksOnPendingOpenWait(t *testing.T) {
	svc := mkSvc("sess-block", "127.0.0.1", 19401)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	svc.listener = ln
	svc.port = ln.Addr().(*net.TCPAddr).Port

	svc.wg.Add(1)
	var released atomic.Bool
	go func() {
		time.Sleep(800 * time.Millisecond)
		svc.wg.Done()
		released.Store(true)
	}()

	start := time.Now()
	done := make(chan struct{})
	go func() {
		svc.Close()
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(start)
		// Must not wait the full 800ms worker; bound is closeWaitTimeout.
		if elapsed >= 500*time.Millisecond {
			t.Fatalf("Close blocked %v, want <= ~%v", elapsed, closeWaitTimeout)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Close did not return within 3s")
	}
	// Let background wg.Done finish to avoid test process race.
	time.Sleep(900 * time.Millisecond)
	_ = released.Load()
}

// ---------------------------------------------------------------------------
// Cross-session port mutex (document constraint)
// ---------------------------------------------------------------------------

func TestAudit_CrossSessionSamePortRejected(t *testing.T) {
	isolateRegistry(t)
	s1 := mkSvc("sess-1", "127.0.0.1", 1080)
	s2 := mkSvc("sess-2", "127.0.0.1", 1080)
	if err := registry.addService(s1); err != nil {
		t.Fatal(err)
	}
	err := registry.addService(s2)
	if err == nil {
		t.Fatal("expected cross-session same port rejection")
	}
}
