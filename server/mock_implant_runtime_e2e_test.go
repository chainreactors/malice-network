//go:build mockimplant

package main

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	dbmodels "github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/testsupport"
)

func getRuntimeSession(t *testing.T, sessionID string) *core.Session {
	t.Helper()

	session, err := core.Sessions.Get(sessionID)
	if err != nil {
		t.Fatalf("core.Sessions.Get(%q) failed: %v", sessionID, err)
	}
	return session
}

func getRuntimeTask(t *testing.T, sessionID string, taskID uint32) *core.Task {
	t.Helper()

	task := getRuntimeSession(t, sessionID).Tasks.Get(taskID)
	if task == nil {
		t.Fatalf("runtime task %s-%d not found", sessionID, taskID)
	}
	return task
}

func getDBTask(t *testing.T, sessionID string, taskID uint32) *dbmodels.Task {
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

func findTask(tasks []*clientpb.Task, taskID uint32) *clientpb.Task {
	for _, task := range tasks {
		if task.GetTaskId() == taskID {
			return task
		}
	}
	return nil
}

func forceSessionInactive(t *testing.T, sessionID string) (*core.Session, int64) {
	t.Helper()

	session := getRuntimeSession(t, sessionID)
	staleAt := time.Now().Add(-3 * time.Hour).Unix()
	session.SetLastCheckin(staleAt)
	if err := session.Save(); err != nil {
		t.Fatalf("session.Save(inactive) failed: %v", err)
	}
	return session, staleAt
}

func requireDBSessionAliveState(t *testing.T, f *mockRPCFixture, want bool) *clientpb.Session {
	t.Helper()

	session, err := f.h.GetSession(f.mock.SessionID)
	if err != nil {
		t.Fatalf("GetSession(%q) failed: %v", f.mock.SessionID, err)
	}
	if session.GetIsAlive() != want {
		t.Fatalf("db session alive = %v, want %v", session.GetIsAlive(), want)
	}
	return session
}

func TestMockImplantSleepTaskE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	const responseDelay = 150 * time.Millisecond
	f.mock.On(consts.ModuleSleep, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		time.Sleep(responseDelay)
		return send(&implantpb.Spite{
			Body: &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
		})
	})

	before := len(f.mock.RequestsByName(consts.ModuleSleep))
	task, err := f.rpc.Sleep(f.session, &implantpb.Timer{
		Expression: "*/9 * * * * * *",
		Jitter:     0.2,
	})
	if err != nil {
		t.Fatalf("Sleep failed: %v", err)
	}
	if task == nil || task.TaskId == 0 {
		t.Fatalf("Sleep task = %#v, want valid task", task)
	}

	request := waitModuleRequest(t, f.mock, consts.ModuleSleep, before)
	if request.GetSession().GetSessionId() != f.mock.SessionID {
		t.Fatalf("sleep session id = %q, want %q", request.GetSession().GetSessionId(), f.mock.SessionID)
	}
	if request.GetTask().GetTaskId() != task.TaskId {
		t.Fatalf("sleep request task id = %d, want %d", request.GetTask().GetTaskId(), task.TaskId)
	}
	if request.GetSpite().GetSleepRequest().GetExpression() != "*/9 * * * * * *" {
		t.Fatalf("sleep expression = %q, want normalized cron expression", request.GetSpite().GetSleepRequest().GetExpression())
	}
	if request.GetSpite().GetSleepRequest().GetJitter() != 0.2 {
		t.Fatalf("sleep jitter = %v, want 0.2", request.GetSpite().GetSleepRequest().GetJitter())
	}

	waitStart := time.Now()
	content := waitTaskFinish(t, f.rpc, f.mock.SessionID, task.TaskId)
	if time.Since(waitStart) < responseDelay/2 {
		t.Fatalf("WaitTaskFinish returned too early: %v", time.Since(waitStart))
	}
	if content.Task.TaskId != task.TaskId {
		t.Fatalf("wait task id = %d, want %d", content.Task.TaskId, task.TaskId)
	}

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantRealtimeExecuteTaskE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	const (
		firstDelay  = 100 * time.Millisecond
		secondDelay = 100 * time.Millisecond
	)
	f.mock.On(consts.ModuleExecute, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		return testsupport.SendRealisticExecStream(ctx, send, 4242, 0,
			testsupport.MockExecChunk{Delay: firstDelay, Stdout: []byte("alpha")},
			testsupport.MockExecChunk{Delay: secondDelay, Stdout: []byte("omega")},
		)
	})

	before := len(f.mock.RequestsByName(consts.ModuleExecute))
	task, err := f.rpc.Execute(f.session, &implantpb.ExecRequest{
		Path:     "cmd.exe",
		Args:     []string{"/c", "echo", "mock"},
		Output:   true,
		Realtime: true,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if task == nil || task.TaskId == 0 {
		t.Fatalf("Execute task = %#v, want valid task", task)
	}

	request := waitModuleRequest(t, f.mock, consts.ModuleExecute, before)
	if request.GetSpite().GetExecRequest().GetPath() != "cmd.exe" {
		t.Fatalf("execute path = %q, want cmd.exe", request.GetSpite().GetExecRequest().GetPath())
	}
	if !request.GetSpite().GetExecRequest().GetRealtime() {
		t.Fatal("execute request should preserve realtime=true")
	}

	first, err := f.rpc.WaitTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
		Need:      0,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent(first) failed: %v", err)
	}
	if got := string(first.GetSpite().GetExecResponse().GetStdout()); got != "alpha" {
		t.Fatalf("first exec chunk = %q, want alpha", got)
	}

	second, err := f.rpc.WaitTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
		Need:      1,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent(second) failed: %v", err)
	}
	if got := string(second.GetSpite().GetExecResponse().GetStdout()); got != "omega" {
		t.Fatalf("second exec chunk = %q, want omega", got)
	}

	finished := waitTaskFinish(t, f.rpc, f.mock.SessionID, task.TaskId)
	if !finished.Task.Finished {
		t.Fatalf("finished task = %#v, want finished state", finished.Task)
	}
	if finished.Task.Cur != 3 || finished.Task.Total != 3 {
		t.Fatalf("finished task progress = %d/%d, want 3/3", finished.Task.Cur, finished.Task.Total)
	}
	if got := string(finished.GetSpite().GetExecResponse().GetStdout()); got != "" {
		t.Fatalf("finished exec chunk = %q, want empty terminal marker", got)
	}
	if !finished.GetSpite().GetExecResponse().GetEnd() {
		t.Fatal("finished exec chunk should be a terminal end marker")
	}

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantSessionStateTransitionsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	runtimeSession := getRuntimeSession(t, f.mock.SessionID)
	if runtimeSession.WorkDir != f.lib.WorkDir {
		t.Fatalf("initial runtime workdir = %q, want %q", runtimeSession.WorkDir, f.lib.WorkDir)
	}
	if runtimeSession.LastCheckinUnix() == 0 {
		t.Fatal("runtime session should already have a post-register checkin timestamp")
	}
	if runtimeSession.Os == nil || runtimeSession.Os.GetName() != "windows" {
		t.Fatalf("initial runtime os = %#v, want windows", runtimeSession.Os)
	}

	storedSession, err := f.h.GetSession(f.mock.SessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if storedSession.GetWorkdir() != f.lib.WorkDir {
		t.Fatalf("initial stored workdir = %q, want %q", storedSession.GetWorkdir(), f.lib.WorkDir)
	}
	if !storedSession.GetIsAlive() {
		t.Fatal("stored session should already be alive after register + first checkin")
	}
	if storedSession.GetTimer().GetExpression() != "* * * * *" {
		t.Fatalf("initial stored timer = %q, want * * * * *", storedSession.GetTimer().GetExpression())
	}

	sleepBefore := len(f.mock.RequestsByName(consts.ModuleSleep))
	sleepTask, err := f.rpc.Sleep(f.session, &implantpb.Timer{
		Expression: "*/7 * * * * * *",
		Jitter:     0.35,
	})
	if err != nil {
		t.Fatalf("Sleep failed: %v", err)
	}
	sleepRequest := waitModuleRequest(t, f.mock, consts.ModuleSleep, sleepBefore)
	if got := sleepRequest.GetSpite().GetSleepRequest().GetExpression(); got != "*/7 * * * * * *" {
		t.Fatalf("sleep request expression = %q", got)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, sleepTask.TaskId)

	runtimeSession = getRuntimeSession(t, f.mock.SessionID)
	if runtimeSession.Expression != "*/7 * * * * * *" || runtimeSession.Jitter != 0.35 {
		t.Fatalf("runtime timer = %q/%v, want %q/%v", runtimeSession.Expression, runtimeSession.Jitter, "*/7 * * * * * *", 0.35)
	}
	storedSession, err = f.h.GetSession(f.mock.SessionID)
	if err != nil {
		t.Fatalf("GetSession(second) failed: %v", err)
	}
	if storedSession.GetTimer().GetExpression() != "*/7 * * * * * *" || storedSession.GetTimer().GetJitter() != 0.35 {
		t.Fatalf("stored timer = %q/%v, want %q/%v", storedSession.GetTimer().GetExpression(), storedSession.GetTimer().GetJitter(), "*/7 * * * * * *", 0.35)
	}

	nextDir := f.lib.WorkDir + `\state-check`
	cdBefore := len(f.mock.RequestsByName(consts.ModuleCd))
	cdTask, err := f.rpc.Cd(f.session, &implantpb.Request{
		Name:  consts.ModuleCd,
		Input: nextDir,
	})
	if err != nil {
		t.Fatalf("Cd failed: %v", err)
	}
	cdRequest := waitModuleRequest(t, f.mock, consts.ModuleCd, cdBefore)
	if got := cdRequest.GetSpite().GetRequest().GetInput(); got != nextDir {
		t.Fatalf("cd request input = %q, want %q", got, nextDir)
	}
	cdContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, cdTask.TaskId)
	if got := cdContent.GetSpite().GetResponse().GetOutput(); got != normalizePath(nextDir) {
		t.Fatalf("cd output = %q, want %q", got, normalizePath(nextDir))
	}

	runtimeSession = getRuntimeSession(t, f.mock.SessionID)
	if runtimeSession.WorkDir != normalizePath(nextDir) {
		t.Fatalf("runtime workdir after cd = %q, want %q", runtimeSession.WorkDir, normalizePath(nextDir))
	}
	storedSession, err = f.h.GetSession(f.mock.SessionID)
	if err != nil {
		t.Fatalf("GetSession(third) failed: %v", err)
	}
	if storedSession.GetWorkdir() != normalizePath(nextDir) {
		t.Fatalf("stored workdir after cd = %q, want %q", storedSession.GetWorkdir(), normalizePath(nextDir))
	}

	infoDir := `C:\Users\analyst\triage`
	infoPath := infoDir + `\agent.exe`
	f.mock.On(consts.ModuleSysInfo, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		return send(&implantpb.Spite{
			Body: &implantpb.Spite_Sysinfo{Sysinfo: &implantpb.SysInfo{
				Filepath:    infoPath,
				Workdir:     infoDir,
				IsPrivilege: false,
				Os: &implantpb.Os{
					Name:     "WINDOWS",
					Arch:     "x86_64",
					Hostname: "ops-host",
					Username: "analyst",
				},
				Process: &implantpb.Process{
					Name:  "agent.exe",
					Pid:   9001,
					Path:  infoPath,
					Owner: `OPS\analyst`,
				},
			}},
		})
	})

	infoBefore := len(f.mock.RequestsByName(consts.ModuleSysInfo))
	infoTask, err := f.rpc.Info(f.session, &implantpb.Request{Name: consts.ModuleSysInfo})
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleSysInfo, infoBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, infoTask.TaskId)

	runtimeSession = getRuntimeSession(t, f.mock.SessionID)
	if runtimeSession.WorkDir != infoDir {
		t.Fatalf("runtime workdir after info = %q, want %q", runtimeSession.WorkDir, infoDir)
	}
	if runtimeSession.Filepath != infoPath {
		t.Fatalf("runtime filepath after info = %q, want %q", runtimeSession.Filepath, infoPath)
	}
	if runtimeSession.Os.GetName() != "windows" || runtimeSession.Os.GetArch() != "x64" {
		t.Fatalf("runtime os after info = %#v, want normalized windows/x64", runtimeSession.Os)
	}
	if runtimeSession.Process.GetPid() != 9001 {
		t.Fatalf("runtime process pid after info = %d, want 9001", runtimeSession.Process.GetPid())
	}

	storedSession, err = f.h.GetSession(f.mock.SessionID)
	if err != nil {
		t.Fatalf("GetSession(fourth) failed: %v", err)
	}
	if storedSession.GetWorkdir() != infoDir {
		t.Fatalf("stored workdir after info = %q, want %q", storedSession.GetWorkdir(), infoDir)
	}
	if storedSession.GetFilepath() != infoPath {
		t.Fatalf("stored filepath after info = %q, want %q", storedSession.GetFilepath(), infoPath)
	}
	if storedSession.GetOs().GetName() != "windows" || storedSession.GetOs().GetArch() != "x64" {
		t.Fatalf("stored os after info = %#v, want normalized windows/x64", storedSession.GetOs())
	}
}

func TestMockImplantSingleResponseTaskStateE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	const responseDelay = 200 * time.Millisecond
	f.mock.On(consts.ModulePing, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		time.Sleep(responseDelay)
		return send(&implantpb.Spite{
			Body: &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: req.GetSpite().GetPing().GetNonce()}},
		})
	})

	pingBefore := len(f.mock.RequestsByName(consts.ModulePing))
	task, err := f.rpc.Ping(f.session, &implantpb.Ping{Nonce: 101})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if task.GetTimeout() {
		t.Fatal("new single-response task should not be timed out immediately")
	}
	if task.GetCreatedAt() == 0 {
		t.Fatal("new single-response task should have created_at set")
	}

	waitModuleRequest(t, f.mock, consts.ModulePing, pingBefore)

	runtimeTask := getRuntimeTask(t, f.mock.SessionID, task.TaskId)
	if cur, total := runtimeTask.Progress(); cur != 0 || total != 1 {
		t.Fatalf("runtime task progress before response = %d/%d, want 0/1", cur, total)
	}
	if runtimeTask.Finished() {
		t.Fatal("runtime task should not be finished before response")
	}
	if runtimeTask.CreatedAt.IsZero() {
		t.Fatal("runtime task should record creation time")
	}
	if runtimeTask.Timeout() {
		t.Fatal("runtime task should not be timed out before response")
	}

	dbTask := getDBTask(t, f.mock.SessionID, task.TaskId)
	if dbTask.Cur != 0 || dbTask.Total != 1 {
		t.Fatalf("db task progress before response = %d/%d, want 0/1", dbTask.Cur, dbTask.Total)
	}
	if !dbTask.FinishTime.IsZero() {
		t.Fatalf("db task finish time before response = %v, want zero", dbTask.FinishTime)
	}

	tasksBeforeFinish, err := f.rpc.GetTasks(context.Background(), &clientpb.TaskRequest{SessionId: f.mock.SessionID})
	if err != nil {
		t.Fatalf("GetTasks(before finish) failed: %v", err)
	}
	listedTask := findTask(tasksBeforeFinish.GetTasks(), task.TaskId)
	if listedTask == nil {
		t.Fatalf("GetTasks(before finish) did not return task %d", task.TaskId)
	}
	if listedTask.GetFinished() {
		t.Fatal("listed task should not be finished before response")
	}

	content := waitTaskFinish(t, f.rpc, f.mock.SessionID, task.TaskId)
	if content.GetTask().GetTimeout() {
		t.Fatal("finished single-response task should not be timed out immediately")
	}
	if !content.GetTask().GetFinished() {
		t.Fatal("finished single-response task should report finished=true")
	}
	if content.GetTask().GetFinishedAt() == 0 {
		t.Fatal("finished single-response task should have finished_at set")
	}
	if content.GetSpite().GetPing().GetNonce() != 101 {
		t.Fatalf("finished ping nonce = %d, want 101", content.GetSpite().GetPing().GetNonce())
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return runtimeTask.IsClosed() && runtimeTask.Ctx.Err() != nil
	}, "single-response task close")
	if cur, total := runtimeTask.Progress(); cur != 1 || total != 1 {
		t.Fatalf("runtime task progress after finish = %d/%d, want 1/1", cur, total)
	}
	if !runtimeTask.Finished() {
		t.Fatal("runtime task should be finished after response")
	}
	if runtimeTask.FinishedAt.IsZero() {
		t.Fatal("runtime task should record finish time")
	}

	dbTask = getDBTask(t, f.mock.SessionID, task.TaskId)
	if dbTask.Cur != 1 || dbTask.Total != 1 {
		t.Fatalf("db task progress after finish = %d/%d, want 1/1", dbTask.Cur, dbTask.Total)
	}
	if dbTask.FinishTime.IsZero() {
		t.Fatal("db task should have finish_time set after response")
	}

	taskContent, err := f.rpc.GetTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
		Need:      0,
	})
	if err != nil {
		t.Fatalf("GetTaskContent failed: %v", err)
	}
	if taskContent.GetSpite().GetPing().GetNonce() != 101 {
		t.Fatalf("GetTaskContent nonce = %d, want 101", taskContent.GetSpite().GetPing().GetNonce())
	}
}

func TestMockImplantStreamingTaskStateAndRecoveryE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	const (
		firstDelay  = 120 * time.Millisecond
		secondDelay = 120 * time.Millisecond
	)
	f.mock.On(consts.ModuleExecute, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		return testsupport.SendRealisticExecStream(ctx, send, 4242, 0,
			testsupport.MockExecChunk{Delay: firstDelay, Stdout: []byte("alpha")},
			testsupport.MockExecChunk{Delay: secondDelay, Stdout: []byte("omega")},
		)
	})

	execBefore := len(f.mock.RequestsByName(consts.ModuleExecute))
	task, err := f.rpc.Execute(f.session, &implantpb.ExecRequest{
		Path:     "cmd.exe",
		Args:     []string{"/c", "echo", "stream"},
		Output:   true,
		Realtime: true,
	})
	if err != nil {
		t.Fatalf("Execute realtime failed: %v", err)
	}
	if task.GetTimeout() {
		t.Fatal("new streaming task should not be timed out immediately")
	}
	if task.GetCreatedAt() == 0 {
		t.Fatal("new streaming task should have created_at set")
	}

	waitModuleRequest(t, f.mock, consts.ModuleExecute, execBefore)

	runtimeTask := getRuntimeTask(t, f.mock.SessionID, task.TaskId)
	if cur, total := runtimeTask.Progress(); cur != 0 || total != -1 {
		t.Fatalf("runtime streaming task progress before response = %d/%d, want 0/-1", cur, total)
	}
	if runtimeTask.Finished() {
		t.Fatal("runtime streaming task should not be finished before callbacks")
	}
	if runtimeTask.Timeout() {
		t.Fatal("runtime streaming task should not be timed out before callbacks")
	}

	dbTask := getDBTask(t, f.mock.SessionID, task.TaskId)
	if dbTask.Cur != 0 || dbTask.Total != -1 {
		t.Fatalf("db streaming task progress before response = %d/%d, want 0/-1", dbTask.Cur, dbTask.Total)
	}
	if !dbTask.FinishTime.IsZero() {
		t.Fatalf("db streaming task finish time before response = %v, want zero", dbTask.FinishTime)
	}

	first, err := f.rpc.WaitTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
		Need:      0,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent(first) failed: %v", err)
	}
	if got := string(first.GetSpite().GetExecResponse().GetStdout()); got != "alpha" {
		t.Fatalf("first exec stdout = %q, want alpha", got)
	}
	if cur, total := runtimeTask.Progress(); cur != 1 || total != -1 {
		t.Fatalf("runtime streaming task progress after first callback = %d/%d, want 1/-1", cur, total)
	}
	if runtimeTask.Finished() {
		t.Fatal("runtime streaming task should stay unfinished after first callback")
	}

	dbTask = getDBTask(t, f.mock.SessionID, task.TaskId)
	if dbTask.Cur != 1 || dbTask.Total != -1 {
		t.Fatalf("db streaming task progress after first callback = %d/%d, want 1/-1", dbTask.Cur, dbTask.Total)
	}
	if !dbTask.FinishTime.IsZero() {
		t.Fatal("db streaming task should not have finish_time after first callback")
	}

	second, err := f.rpc.WaitTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
		Need:      1,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent(second) failed: %v", err)
	}
	if got := string(second.GetSpite().GetExecResponse().GetStdout()); got != "omega" {
		t.Fatalf("second exec stdout = %q, want omega", got)
	}

	finished := waitTaskFinish(t, f.rpc, f.mock.SessionID, task.TaskId)
	if !finished.GetTask().GetFinished() {
		t.Fatal("finished streaming task should report finished=true")
	}
	if finished.GetTask().GetCur() != 3 || finished.GetTask().GetTotal() != 3 {
		t.Fatalf("finished streaming task progress = %d/%d, want 3/3", finished.GetTask().GetCur(), finished.GetTask().GetTotal())
	}
	if finished.GetTask().GetFinishedAt() == 0 {
		t.Fatal("finished streaming task should have finished_at set")
	}
	if finished.GetTask().GetTimeout() {
		t.Fatal("finished streaming task should not be timed out immediately")
	}
	if got := string(finished.GetSpite().GetExecResponse().GetStdout()); got != "" {
		t.Fatalf("finished streaming terminal chunk = %q, want empty end marker", got)
	}
	if !finished.GetSpite().GetExecResponse().GetEnd() {
		t.Fatal("finished streaming task should return a terminal end marker")
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return runtimeTask.IsClosed() && runtimeTask.Ctx.Err() != nil
	}, "streaming task close")
	if cur, total := runtimeTask.Progress(); cur != 3 || total != 3 {
		t.Fatalf("runtime streaming task progress after finish = %d/%d, want 3/3", cur, total)
	}
	if !runtimeTask.Finished() {
		t.Fatal("runtime streaming task should be finished")
	}

	dbTask = getDBTask(t, f.mock.SessionID, task.TaskId)
	if dbTask.Cur != 3 || dbTask.Total != 3 {
		t.Fatalf("db streaming task progress after finish = %d/%d, want 3/3", dbTask.Cur, dbTask.Total)
	}
	if dbTask.FinishTime.IsZero() {
		t.Fatal("db streaming task should have finish_time set after completion")
	}

	allContent, err := f.rpc.GetAllTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
	})
	if err != nil {
		t.Fatalf("GetAllTaskContent failed: %v", err)
	}
	if len(allContent.GetSpites()) != 3 {
		t.Fatalf("GetAllTaskContent count = %d, want 3", len(allContent.GetSpites()))
	}
	lastSpite := allContent.GetSpites()[len(allContent.GetSpites())-1]
	if !lastSpite.GetExecResponse().GetEnd() {
		t.Fatalf("last exec spite = %#v, want terminal end marker", lastSpite.GetExecResponse())
	}

	runtimeSession := getRuntimeSession(t, f.mock.SessionID)
	runtimeSession.Tasks.Remove(task.TaskId)

	recoveredFirst, err := f.rpc.GetTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
		Need:      0,
	})
	if err != nil {
		t.Fatalf("GetTaskContent(recovered first) failed: %v", err)
	}
	if got := string(recoveredFirst.GetSpite().GetExecResponse().GetStdout()); got != "alpha" {
		t.Fatalf("recovered first stdout = %q, want alpha", got)
	}

	recoveredSecond, err := f.rpc.GetTaskContent(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
		Need:      1,
	})
	if err != nil {
		t.Fatalf("GetTaskContent(recovered second) failed: %v", err)
	}
	if got := string(recoveredSecond.GetSpite().GetExecResponse().GetStdout()); got != "omega" {
		t.Fatalf("recovered second stdout = %q, want omega", got)
	}

	recoveredFinished, err := f.rpc.WaitTaskFinish(context.Background(), &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
	})
	if err != nil {
		t.Fatalf("WaitTaskFinish(recovered) failed: %v", err)
	}
	if !recoveredFinished.GetTask().GetFinished() {
		t.Fatal("recovered streaming task should stay finished")
	}
	if got := string(recoveredFinished.GetSpite().GetExecResponse().GetStdout()); got != "" {
		t.Fatalf("recovered final stdout = %q, want empty terminal marker", got)
	}
	if !recoveredFinished.GetSpite().GetExecResponse().GetEnd() {
		t.Fatal("recovered final spite should stay a terminal end marker")
	}

	tasksAfterFinish, err := f.rpc.GetTasks(context.Background(), &clientpb.TaskRequest{SessionId: f.mock.SessionID})
	if err != nil {
		t.Fatalf("GetTasks(after finish) failed: %v", err)
	}
	listedTask := findTask(tasksAfterFinish.GetTasks(), task.TaskId)
	if listedTask == nil {
		t.Fatalf("GetTasks(after finish) did not return task %d", task.TaskId)
	}
	if !listedTask.GetFinished() || listedTask.GetCur() != 3 || listedTask.GetTotal() != 3 {
		t.Fatalf("listed task after finish = %#v, want finished 3/3", listedTask)
	}
}

func TestMockImplantDeadSweepKeepsPendingSingleResponseTaskAlive(t *testing.T) {
	f := newMockRPCFixture(t)

	const responseDelay = 400 * time.Millisecond
	f.mock.On(consts.ModulePing, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		time.Sleep(responseDelay)
		return send(&implantpb.Spite{
			Body: &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: req.GetSpite().GetPing().GetNonce()}},
		})
	})

	pingBefore := len(f.mock.RequestsByName(consts.ModulePing))
	task, err := f.rpc.Ping(f.session, &implantpb.Ping{Nonce: 4242})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	waitModuleRequest(t, f.mock, consts.ModulePing, pingBefore)

	runtimeTask := getRuntimeTask(t, f.mock.SessionID, task.TaskId)
	_, staleAt := forceSessionInactive(t, f.mock.SessionID)
	core.SweepInactiveSessions()

	runtimeSession := getRuntimeSession(t, f.mock.SessionID)
	if !runtimeSession.IsMarkedDead() {
		t.Fatal("runtime session should be marked dead after inactive sweep")
	}
	if runtimeSession.Ctx.Err() != nil {
		t.Fatal("runtime session should stay alive while unfinished tasks exist")
	}
	if runtimeTask.Ctx.Err() != nil {
		t.Fatal("pending task context should stay alive across dead sweep")
	}
	requireDBSessionAliveState(t, f, false)

	finished := waitTaskFinish(t, f.rpc, f.mock.SessionID, task.TaskId)
	if got := finished.GetSpite().GetPing().GetNonce(); got != 4242 {
		t.Fatalf("finished ping nonce = %d, want 4242", got)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		session, err := core.Sessions.Get(f.mock.SessionID)
		return err == nil && !session.IsMarkedDead() && session.LastCheckinUnix() > staleAt
	}, "session reborn after late ping response")

	requireDBSessionAliveState(t, f, true)
}

func TestMockImplantDeadSweepKeepsPendingStreamingTaskAlive(t *testing.T) {
	f := newMockRPCFixture(t)

	releaseFinal := make(chan struct{})
	f.mock.On(consts.ModuleExecute, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		if err := send(&implantpb.Spite{
			Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
				Stdout: []byte("alpha"),
				Pid:    7001,
				End:    false,
			}},
		}); err != nil {
			return err
		}

		select {
		case <-releaseFinal:
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return context.DeadlineExceeded
		}
		return testsupport.SendRealisticExecStream(ctx, send, 7001, 0,
			testsupport.MockExecChunk{Stdout: []byte("omega")},
		)
	})

	execBefore := len(f.mock.RequestsByName(consts.ModuleExecute))
	task, err := f.rpc.Execute(f.session, &implantpb.ExecRequest{
		Path:     "cmd.exe",
		Args:     []string{"/c", "echo", "edge"},
		Output:   true,
		Realtime: true,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	waitModuleRequest(t, f.mock, consts.ModuleExecute, execBefore)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	first, err := f.rpc.WaitTaskContent(ctx, &clientpb.Task{
		SessionId: f.mock.SessionID,
		TaskId:    task.TaskId,
		Need:      0,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent(first) failed: %v", err)
	}
	if got := string(first.GetSpite().GetExecResponse().GetStdout()); got != "alpha" {
		t.Fatalf("first exec stdout = %q, want alpha", got)
	}

	runtimeTask := getRuntimeTask(t, f.mock.SessionID, task.TaskId)
	_, staleAt := forceSessionInactive(t, f.mock.SessionID)
	core.SweepInactiveSessions()

	runtimeSession := getRuntimeSession(t, f.mock.SessionID)
	if !runtimeSession.IsMarkedDead() {
		t.Fatal("runtime session should be marked dead while streaming task is pending")
	}
	if runtimeSession.Ctx.Err() != nil {
		t.Fatal("runtime session should stay alive while streaming task is unfinished")
	}
	if runtimeTask.Ctx.Err() != nil {
		t.Fatal("streaming task context should survive dead sweep")
	}
	requireDBSessionAliveState(t, f, false)

	close(releaseFinal)
	finished := waitTaskFinish(t, f.rpc, f.mock.SessionID, task.TaskId)
	if got := string(finished.GetSpite().GetExecResponse().GetStdout()); got != "" {
		t.Fatalf("finished exec stdout = %q, want empty terminal marker", got)
	}
	if !finished.GetTask().GetFinished() {
		t.Fatal("streaming task should finish after final callback")
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		session, err := core.Sessions.Get(f.mock.SessionID)
		return err == nil && !session.IsMarkedDead() && session.LastCheckinUnix() > staleAt
	}, "session reborn after late streaming response")

	requireDBSessionAliveState(t, f, true)
}

func TestMockImplantIdleDeadSessionRemovedAndCheckinReborns(t *testing.T) {
	f := newMockRPCFixture(t)

	staleSession, staleAt := forceSessionInactive(t, f.mock.SessionID)
	core.SweepInactiveSessions()

	if _, err := core.Sessions.Get(f.mock.SessionID); err == nil {
		t.Fatal("idle dead session should be removed from runtime session map")
	}
	if staleSession.Ctx.Err() == nil {
		t.Fatal("removed dead session context should be cancelled")
	}
	requireDBSessionAliveState(t, f, false)

	if err := f.mock.Checkin(); err != nil {
		t.Fatalf("mock implant checkin failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		session, err := core.Sessions.Get(f.mock.SessionID)
		return err == nil && !session.IsMarkedDead() && session.LastCheckinUnix() > staleAt
	}, "session reborn after checkin")

	rebornSession := getRuntimeSession(t, f.mock.SessionID)
	if rebornSession.Ctx.Err() != nil {
		t.Fatal("reborn session should have a live runtime context")
	}
	requireDBSessionAliveState(t, f, true)
}
