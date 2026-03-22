//go:build realimplant

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	dbmodels "github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/testsupport"
	"google.golang.org/grpc/metadata"
)

type realRPCFixture struct {
	h       *testsupport.ControlPlaneHarness
	implant *testsupport.RealImplant
	rpc     clientrpc.MaliceRPCClient
	session context.Context
}

func newRealRPCFixture(t *testing.T) *realRPCFixture {
	t.Helper()

	testsupport.RequireRealImplantEnv(t)

	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("real-listener-%d", time.Now().UnixNano())
	pipelineName := fmt.Sprintf("real-pipe-%d", time.Now().UnixNano())
	implant := testsupport.NewRealImplant(t, h, testsupport.NewRealTCPPipeline(t, listenerName, pipelineName))
	if err := implant.Start(t); err != nil {
		t.Fatalf("real implant start failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	session := mustRealRuntimeSession(t, implant.SessionID)
	requireModulePresent(t, session.Modules, consts.ModulePwd)
	requireModulePresent(t, session.Modules, consts.ModuleExecute)

	return &realRPCFixture{
		h:       h,
		implant: implant,
		rpc:     clientrpc.NewMaliceRPCClient(conn),
		session: metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
			"session_id", implant.SessionID,
			"callee", consts.CalleeCMD,
		)),
	}
}

func mustRealRuntimeSession(t *testing.T, sessionID string) *core.Session {
	t.Helper()

	session, err := core.Sessions.Get(sessionID)
	if err != nil {
		t.Fatalf("core.Sessions.Get(%q) failed: %v", sessionID, err)
	}
	return session
}

func mustRealRuntimeTask(t *testing.T, sessionID string, taskID uint32) *core.Task {
	t.Helper()

	task := mustRealRuntimeSession(t, sessionID).Tasks.Get(taskID)
	if task == nil {
		t.Fatalf("runtime task %s-%d not found", sessionID, taskID)
	}
	return task
}

func mustRealDBTask(t *testing.T, sessionID string, taskID uint32) *dbmodels.Task {
	t.Helper()

	task, err := db.GetTaskBySessionAndSeq(sessionID, taskID)
	if err != nil {
		t.Fatalf("GetTaskBySessionAndSeq(%q,%d) failed: %v", sessionID, taskID, err)
	}
	if task == nil {
		t.Fatalf("db task %s-%d not found", sessionID, taskID)
	}
	return task
}

func waitRealTaskFinish(t *testing.T, rpc clientrpc.MaliceRPCClient, sessionID string, taskID uint32) *clientpb.TaskContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	content, err := rpc.WaitTaskFinish(ctx, &clientpb.Task{
		SessionId: sessionID,
		TaskId:    taskID,
	})
	if err != nil {
		t.Fatalf("WaitTaskFinish(%d) failed: %v", taskID, err)
	}
	if content == nil || content.Task == nil || content.Spite == nil {
		t.Fatalf("WaitTaskFinish(%d) returned incomplete content: %#v", taskID, content)
	}
	return content
}

func markRealSessionStale(t *testing.T, sessionID string) int64 {
	t.Helper()

	session := mustRealRuntimeSession(t, sessionID)
	staleAt := time.Now().Add(-10 * time.Minute).Unix()
	session.SetLastCheckin(staleAt)
	if err := session.Save(); err != nil {
		t.Fatalf("session.Save(stale) failed: %v", err)
	}
	return staleAt
}

func requireRealDBSessionAlive(t *testing.T, h *testsupport.ControlPlaneHarness, sessionID string, want bool) *clientpb.Session {
	t.Helper()

	session, err := h.GetSession(sessionID)
	if err != nil {
		t.Fatalf("GetSession(%q) failed: %v", sessionID, err)
	}
	if session.GetIsAlive() != want {
		t.Fatalf("db session alive = %v, want %v", session.GetIsAlive(), want)
	}
	return session
}

func requireModulePresent(t *testing.T, modules []string, want string) {
	t.Helper()

	for _, module := range modules {
		if module == want {
			return
		}
	}
	t.Fatalf("module list %v does not contain %q", modules, want)
}

func waitRealActiveConnection(t *testing.T, sessionID string) {
	t.Helper()

	testsupport.WaitForCondition(t, 10*time.Second, func() bool {
		conn := core.Connections.Get(sessionID)
		return conn != nil && conn.IsAlive()
	}, "real implant active connection "+sessionID)
}

func waitRealPostRegisterCheckin(t *testing.T, sessionID string) {
	t.Helper()

	initial := mustRealRuntimeSession(t, sessionID).LastCheckinUnix()
	testsupport.WaitForCondition(t, 12*time.Second, func() bool {
		return mustRealRuntimeSession(t, sessionID).LastCheckinUnix() > initial
	}, "real implant post-register checkin "+sessionID)
}

func normalizeWindowsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.ReplaceAll(path, "/", `\`)
	path = filepath.Clean(path)
	if len(path) == 2 && strings.HasSuffix(path, ":") {
		path += `\`
	}
	return strings.ToLower(path)
}

func enableRealKeepalive(t *testing.T, f *realRPCFixture, enabled bool) {
	t.Helper()

	task, err := f.rpc.Keepalive(f.session, &implantpb.CommonBody{
		BoolArray: []bool{enabled},
	})
	if err != nil {
		t.Fatalf("Keepalive(%v) failed: %v", enabled, err)
	}
	waitRealTaskFinish(t, f.rpc, f.implant.SessionID, task.TaskId)
	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return mustRealRuntimeSession(t, f.implant.SessionID).IsKeepaliveEnabled() == enabled
	}, fmt.Sprintf("keepalive=%v runtime state", enabled))
}

func TestRealImplantBasicRPCStateE2E(t *testing.T) {
	f := newRealRPCFixture(t)
	waitRealActiveConnection(t, f.implant.SessionID)
	waitRealPostRegisterCheckin(t, f.implant.SessionID)

	runtimeSession := mustRealRuntimeSession(t, f.implant.SessionID)
	if runtimeSession.WorkDir == "" {
		t.Fatal("registered runtime session should include a non-empty workdir")
	}
	if runtimeSession.Os == nil || runtimeSession.Os.GetName() == "" {
		t.Fatalf("registered runtime session os = %#v, want non-empty", runtimeSession.Os)
	}

	storedSession, err := f.h.GetSession(f.implant.SessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if storedSession.GetWorkdir() == "" {
		t.Fatal("stored session should persist a non-empty workdir")
	}

	sleepTask, err := f.rpc.Sleep(f.session, &implantpb.Timer{
		Expression: "*/7 * * * * * *",
		Jitter:     0.15,
	})
	if err != nil {
		t.Fatalf("Sleep failed: %v", err)
	}
	waitRealTaskFinish(t, f.rpc, f.implant.SessionID, sleepTask.TaskId)

	runtimeSession = mustRealRuntimeSession(t, f.implant.SessionID)
	if runtimeSession.Expression != "*/7 * * * * * *" || runtimeSession.Jitter != 0.15 {
		t.Fatalf("runtime timer = %q/%v, want %q/%v", runtimeSession.Expression, runtimeSession.Jitter, "*/7 * * * * * *", 0.15)
	}
	storedSession, err = f.h.GetSession(f.implant.SessionID)
	if err != nil {
		t.Fatalf("GetSession(after sleep) failed: %v", err)
	}
	if storedSession.GetTimer().GetExpression() != "*/7 * * * * * *" || storedSession.GetTimer().GetJitter() != 0.15 {
		t.Fatalf("stored timer = %q/%v, want %q/%v", storedSession.GetTimer().GetExpression(), storedSession.GetTimer().GetJitter(), "*/7 * * * * * *", 0.15)
	}

	enableRealKeepalive(t, f, true)

	waitRealActiveConnection(t, f.implant.SessionID)
	pwdTask, err := f.rpc.Pwd(f.session, &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("Pwd failed: %v", err)
	}
	pwdContent := waitRealTaskFinish(t, f.rpc, f.implant.SessionID, pwdTask.TaskId)
	pwdOutput := strings.TrimSpace(pwdContent.GetSpite().GetResponse().GetOutput())
	if pwdOutput == "" {
		t.Fatal("pwd output should not be empty")
	}
	runtimeSession = mustRealRuntimeSession(t, f.implant.SessionID)
	if normalizeWindowsPath(pwdOutput) != normalizeWindowsPath(runtimeSession.WorkDir) {
		t.Fatalf("pwd output = %q, want runtime workdir %q", pwdOutput, runtimeSession.WorkDir)
	}

	enableRealKeepalive(t, f, false)
}

func TestRealImplantDeadSweepKeepsPendingStreamingTaskAlive(t *testing.T) {
	f := newRealRPCFixture(t)
	waitRealActiveConnection(t, f.implant.SessionID)
	waitRealPostRegisterCheckin(t, f.implant.SessionID)

	enableRealKeepalive(t, f, true)

	waitRealActiveConnection(t, f.implant.SessionID)
	task, err := f.rpc.Execute(f.session, &implantpb.ExecRequest{
		Path: "cmd.exe",
		Args: []string{
			"/c",
			"ping -n 2 127.0.0.1 > nul & echo alpha & ping -n 3 127.0.0.1 > nul & echo omega",
		},
		Output:   true,
		Realtime: true,
	})
	if err != nil {
		t.Fatalf("Execute realtime failed: %v", err)
	}

	runtimeTask := mustRealRuntimeTask(t, f.implant.SessionID, task.TaskId)
	if cur, total := runtimeTask.Progress(); cur != 0 || total != -1 {
		t.Fatalf("runtime task progress before first chunk = %d/%d, want 0/-1", cur, total)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	first, err := f.rpc.WaitTaskContent(ctx, &clientpb.Task{
		SessionId: f.implant.SessionID,
		TaskId:    task.TaskId,
		Need:      0,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent(first) failed: %v", err)
	}
	firstChunk := string(first.GetSpite().GetExecResponse().GetStdout())
	if !strings.Contains(strings.ToLower(firstChunk), "alpha") {
		t.Fatalf("first exec chunk = %q, want alpha", firstChunk)
	}

	if cur, total := runtimeTask.Progress(); cur != 1 || total != -1 {
		t.Fatalf("runtime task progress after first chunk = %d/%d, want 1/-1", cur, total)
	}

	dbTask := mustRealDBTask(t, f.implant.SessionID, task.TaskId)
	if dbTask.Cur != 1 || dbTask.Total != -1 {
		t.Fatalf("db task progress after first chunk = %d/%d, want 1/-1", dbTask.Cur, dbTask.Total)
	}
	if !dbTask.FinishTime.IsZero() {
		t.Fatalf("db task finish time before final chunk = %v, want zero", dbTask.FinishTime)
	}

	staleAt := markRealSessionStale(t, f.implant.SessionID)
	core.SweepInactiveSessions()

	runtimeSession := mustRealRuntimeSession(t, f.implant.SessionID)
	if !runtimeSession.IsMarkedDead() {
		t.Fatal("runtime session should be marked dead after inactive sweep")
	}
	if runtimeSession.Ctx.Err() != nil {
		t.Fatal("runtime session should stay alive while the streaming task is unfinished")
	}
	if runtimeTask.Ctx.Err() != nil {
		t.Fatal("pending streaming task context should stay alive across dead sweep")
	}
	requireRealDBSessionAlive(t, f.h, f.implant.SessionID, false)

	finished := waitRealTaskFinish(t, f.rpc, f.implant.SessionID, task.TaskId)
	finalChunk := string(finished.GetSpite().GetExecResponse().GetStdout())
	if strings.TrimSpace(finalChunk) != "" {
		t.Fatalf("terminal exec chunk = %q, want empty end marker", finalChunk)
	}
	if !finished.GetTask().GetFinished() {
		t.Fatal("streaming task should report finished after the final chunk")
	}

	allContent, err := f.rpc.GetAllTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.implant.SessionID,
		TaskId:    task.TaskId,
	})
	if err != nil {
		t.Fatalf("GetAllTaskContent failed: %v", err)
	}
	if len(allContent.GetSpites()) < 2 {
		t.Fatalf("streaming task content count = %d, want at least 2", len(allContent.GetSpites()))
	}
	foundOmega := false
	for _, spite := range allContent.GetSpites() {
		if strings.Contains(strings.ToLower(string(spite.GetExecResponse().GetStdout())), "omega") {
			foundOmega = true
			break
		}
	}
	if !foundOmega {
		t.Fatalf("streaming task content = %#v, want one chunk containing omega", allContent.GetSpites())
	}
	lastSpite := allContent.GetSpites()[len(allContent.GetSpites())-1]
	if !lastSpite.GetExecResponse().GetEnd() {
		t.Fatalf("last exec spite = %#v, want terminal end marker", lastSpite.GetExecResponse())
	}

	testsupport.WaitForCondition(t, 10*time.Second, func() bool {
		session, err := core.Sessions.Get(f.implant.SessionID)
		return err == nil && !session.IsMarkedDead() && session.LastCheckinUnix() > staleAt
	}, "real implant session reborn after late streaming response")

	requireRealDBSessionAlive(t, f.h, f.implant.SessionID, true)

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return runtimeTask.IsClosed() && runtimeTask.Ctx.Err() != nil
	}, "real implant streaming task close")
	wantTotal := len(allContent.GetSpites())
	if cur, total := runtimeTask.Progress(); cur != wantTotal || total != wantTotal {
		t.Fatalf("runtime task progress after finish = %d/%d, want %d/%d", cur, total, wantTotal, wantTotal)
	}

	dbTask = mustRealDBTask(t, f.implant.SessionID, task.TaskId)
	if dbTask.Cur != wantTotal || dbTask.Total != wantTotal {
		t.Fatalf("db task progress after finish = %d/%d, want %d/%d", dbTask.Cur, dbTask.Total, wantTotal, wantTotal)
	}
	if dbTask.FinishTime.IsZero() {
		t.Fatal("db task should record finish_time after the final chunk")
	}
}

// TestRealImplantSecureKeyExchangeE2E verifies the full age key exchange flow:
//  1. Implant starts with empty age keys (cold start, secure feature enabled)
//  2. Server has a secure pipeline that triggers key exchange on first registration
//  3. Key exchange completes: server sends KeyExchangeRequest, implant responds
//  4. Implant reconnects with new keys
//  5. Commands (pwd) execute correctly over the age-encrypted channel
//  6. HMAC signature on KeyExchangeRequest is verified by implant
func TestRealImplantSecureKeyExchangeE2E(t *testing.T) {
	testsupport.RequireRealImplantEnv(t)

	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("real-secure-listener-%d", time.Now().UnixNano())
	pipelineName := fmt.Sprintf("real-secure-pipe-%d", time.Now().UnixNano())

	// Create a secure TCP pipeline (age key exchange enabled)
	pipeline := testsupport.NewRealSecureTCPPipeline(t, listenerName, pipelineName)
	implant := testsupport.NewRealImplant(t, h, pipeline)
	if err := implant.Start(t); err != nil {
		t.Fatalf("real secure implant start failed: %v", err)
	}

	// Wait for session registration + initial key exchange
	// The server triggers triggerKeyExchange on first Register when secure is enabled
	waitRealActiveConnection(t, implant.SessionID)
	waitRealPostRegisterCheckin(t, implant.SessionID)

	// Verify session was registered with secure mode
	runtimeSession := mustRealRuntimeSession(t, implant.SessionID)
	if runtimeSession.SecureManager == nil {
		t.Fatal("session should have a SecureManager after secure registration")
	}

	// Connect as admin client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	rpc := clientrpc.NewMaliceRPCClient(conn)
	sessionCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", implant.SessionID,
		"callee", consts.CalleeCMD,
	))

	// Enable keepalive for faster command execution
	enableRealKeepalive(t, &realRPCFixture{
		h: h, implant: implant, rpc: rpc, session: sessionCtx,
	}, true)

	waitRealActiveConnection(t, implant.SessionID)

	// Execute pwd command — this goes through age-encrypted channel
	pwdTask, err := rpc.Pwd(sessionCtx, &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("Pwd (post key exchange) failed: %v", err)
	}
	pwdContent := waitRealTaskFinish(t, rpc, implant.SessionID, pwdTask.TaskId)
	pwdOutput := strings.TrimSpace(pwdContent.GetSpite().GetResponse().GetOutput())
	if pwdOutput == "" {
		t.Fatal("pwd output should not be empty after secure key exchange")
	}
	t.Logf("pwd after key exchange: %s", pwdOutput)

	// Verify workdir matches
	runtimeSession = mustRealRuntimeSession(t, implant.SessionID)
	if normalizeWindowsPath(pwdOutput) != normalizeWindowsPath(runtimeSession.WorkDir) {
		t.Fatalf("pwd = %q, want workdir %q", pwdOutput, runtimeSession.WorkDir)
	}

	// Disable keepalive
	enableRealKeepalive(t, &realRPCFixture{
		h: h, implant: implant, rpc: rpc, session: sessionCtx,
	}, false)

	t.Log("secure key exchange E2E test passed: cold start → key exchange → encrypted commands")
}

// TestRealImplantTLSPipelineE2E verifies the implant can connect to a TLS-enabled
// pipeline (self-signed cert with CA verification) and execute commands normally.
// This tests the full certificate chain: pipeline generates CA+cert → profile
// passes CA to implant → implant verifies server cert → encrypted session works.
func TestRealImplantTLSPipelineE2E(t *testing.T) {
	testsupport.RequireRealImplantEnv(t)

	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("real-tls-listener-%d", time.Now().UnixNano())
	pipelineName := fmt.Sprintf("real-tls-pipe-%d", time.Now().UnixNano())

	// Create a TLS-enabled TCP pipeline (self-signed CA + server cert)
	pipeline := testsupport.NewRealTLSTCPPipeline(t, listenerName, pipelineName)
	implant := testsupport.NewRealImplant(t, h, pipeline)
	if err := implant.Start(t); err != nil {
		t.Fatalf("real TLS implant start failed: %v", err)
	}

	waitRealActiveConnection(t, implant.SessionID)
	waitRealPostRegisterCheckin(t, implant.SessionID)

	runtimeSession := mustRealRuntimeSession(t, implant.SessionID)
	if runtimeSession.WorkDir == "" {
		t.Fatal("registered TLS session should include a non-empty workdir")
	}

	// Connect as admin client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	rpc := clientrpc.NewMaliceRPCClient(conn)
	sessionCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", implant.SessionID,
		"callee", consts.CalleeCMD,
	))

	// Enable keepalive
	enableRealKeepalive(t, &realRPCFixture{
		h: h, implant: implant, rpc: rpc, session: sessionCtx,
	}, true)

	waitRealActiveConnection(t, implant.SessionID)

	// Execute pwd over TLS
	pwdTask, err := rpc.Pwd(sessionCtx, &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("Pwd (over TLS) failed: %v", err)
	}
	pwdContent := waitRealTaskFinish(t, rpc, implant.SessionID, pwdTask.TaskId)
	pwdOutput := strings.TrimSpace(pwdContent.GetSpite().GetResponse().GetOutput())
	if pwdOutput == "" {
		t.Fatal("pwd output over TLS should not be empty")
	}
	t.Logf("pwd over TLS: %s", pwdOutput)

	// Verify workdir matches
	runtimeSession = mustRealRuntimeSession(t, implant.SessionID)
	if normalizeWindowsPath(pwdOutput) != normalizeWindowsPath(runtimeSession.WorkDir) {
		t.Fatalf("pwd = %q, want workdir %q", pwdOutput, runtimeSession.WorkDir)
	}

	// Disable keepalive
	enableRealKeepalive(t, &realRPCFixture{
		h: h, implant: implant, rpc: rpc, session: sessionCtx,
	}, false)

	t.Log("TLS pipeline E2E test passed: self-signed CA → TLS handshake → encrypted commands")
}

// TestRealImplantMTLSPipelineE2E verifies mutual TLS: both server and implant
// present certificates signed by the same CA. The server verifies the implant's
// client certificate, and the implant verifies the server's certificate.
//
func TestRealImplantMTLSPipelineE2E(t *testing.T) {
	testsupport.RequireRealImplantEnv(t)

	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("real-mtls-listener-%d", time.Now().UnixNano())
	pipelineName := fmt.Sprintf("real-mtls-pipe-%d", time.Now().UnixNano())

	// Create mTLS pipeline (CA + server cert + client cert)
	pipeline, mtlsCerts := testsupport.NewRealMTLSTCPPipeline(t, listenerName, pipelineName)
	implant := testsupport.NewRealImplant(t, h, pipeline)
	implant.MTLSCerts = mtlsCerts // inject client certs into profile
	if err := implant.Start(t); err != nil {
		t.Fatalf("real mTLS implant start failed: %v", err)
	}

	waitRealActiveConnection(t, implant.SessionID)
	waitRealPostRegisterCheckin(t, implant.SessionID)

	runtimeSession := mustRealRuntimeSession(t, implant.SessionID)
	if runtimeSession.WorkDir == "" {
		t.Fatal("registered mTLS session should include a non-empty workdir")
	}

	// Connect as admin client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	rpc := clientrpc.NewMaliceRPCClient(conn)
	sessionCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", implant.SessionID,
		"callee", consts.CalleeCMD,
	))

	enableRealKeepalive(t, &realRPCFixture{
		h: h, implant: implant, rpc: rpc, session: sessionCtx,
	}, true)

	waitRealActiveConnection(t, implant.SessionID)

	// Execute pwd over mTLS channel
	pwdTask, err := rpc.Pwd(sessionCtx, &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("Pwd (over mTLS) failed: %v", err)
	}
	pwdContent := waitRealTaskFinish(t, rpc, implant.SessionID, pwdTask.TaskId)
	pwdOutput := strings.TrimSpace(pwdContent.GetSpite().GetResponse().GetOutput())
	if pwdOutput == "" {
		t.Fatal("pwd output over mTLS should not be empty")
	}
	t.Logf("pwd over mTLS: %s", pwdOutput)

	enableRealKeepalive(t, &realRPCFixture{
		h: h, implant: implant, rpc: rpc, session: sessionCtx,
	}, false)

	t.Log("mTLS pipeline E2E test passed: mutual certificate verification → encrypted commands")
}
