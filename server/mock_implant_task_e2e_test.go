//go:build mockimplant

package main

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestMockImplantSleepTaskE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	mock := testsupport.NewMockImplant(t, h, h.NewTCPPipeline(t, "mock-implant-task-pipe"))

	const responseDelay = 150 * time.Millisecond
	mock.On(consts.ModuleSleep, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		time.Sleep(responseDelay)
		return send(&implantpb.Spite{
			Body: &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
		})
	})

	if err := mock.Start(); err != nil {
		t.Fatalf("mock implant start failed: %v", err)
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

	rpc := clientrpc.NewMaliceRPCClient(conn)
	sessionCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", mock.SessionID,
		"callee", consts.CalleeCMD,
	))

	task, err := rpc.Sleep(sessionCtx, &implantpb.Timer{
		Expression: "*/9 * * * * * *",
		Jitter:     0.2,
	})
	if err != nil {
		t.Fatalf("Sleep failed: %v", err)
	}
	if task == nil || task.TaskId == 0 {
		t.Fatalf("Sleep task = %#v, want valid task", task)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return len(mock.RequestsByName(consts.ModuleSleep)) == 1
	}, "mock implant to receive sleep request")

	request := mock.LastRequest(consts.ModuleSleep)
	if request == nil {
		t.Fatal("sleep request was not recorded")
	}
	if request.GetSession().GetSessionId() != mock.SessionID {
		t.Fatalf("sleep session id = %q, want %q", request.GetSession().GetSessionId(), mock.SessionID)
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
	content, err := rpc.WaitTaskFinish(context.Background(), &clientpb.Task{
		SessionId: mock.SessionID,
		TaskId:    task.TaskId,
	})
	if err != nil {
		t.Fatalf("WaitTaskFinish failed: %v", err)
	}
	if time.Since(waitStart) < responseDelay/2 {
		t.Fatalf("WaitTaskFinish returned too early: %v", time.Since(waitStart))
	}
	if content == nil || content.Task == nil || content.Spite == nil {
		t.Fatalf("WaitTaskFinish content = %#v, want populated task context", content)
	}
	if content.Task.TaskId != task.TaskId {
		t.Fatalf("wait task id = %d, want %d", content.Task.TaskId, task.TaskId)
	}

	if errs := mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantRealtimeExecuteTaskE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	mock := testsupport.NewMockImplant(t, h, h.NewTCPPipeline(t, "mock-implant-exec-pipe"))

	const (
		firstDelay  = 100 * time.Millisecond
		secondDelay = 100 * time.Millisecond
	)
	mock.On(consts.ModuleExecute, func(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		time.Sleep(firstDelay)
		if err := send(&implantpb.Spite{
			Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
				Stdout: []byte("alpha"),
				End:    false,
			}},
		}); err != nil {
			return err
		}

		time.Sleep(secondDelay)
		return send(&implantpb.Spite{
			Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
				Stdout: []byte("omega"),
				End:    true,
			}},
		})
	})

	if err := mock.Start(); err != nil {
		t.Fatalf("mock implant start failed: %v", err)
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

	rpc := clientrpc.NewMaliceRPCClient(conn)
	sessionCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", mock.SessionID,
		"callee", consts.CalleeCMD,
	))

	task, err := rpc.Execute(sessionCtx, &implantpb.ExecRequest{
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

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return len(mock.RequestsByName(consts.ModuleExecute)) == 1
	}, "mock implant to receive execute request")

	request := mock.LastRequest(consts.ModuleExecute)
	if request == nil {
		t.Fatal("execute request was not recorded")
	}
	if request.GetSpite().GetExecRequest().GetPath() != "cmd.exe" {
		t.Fatalf("execute path = %q, want cmd.exe", request.GetSpite().GetExecRequest().GetPath())
	}
	if !request.GetSpite().GetExecRequest().GetRealtime() {
		t.Fatal("execute request should preserve realtime=true")
	}

	first, err := rpc.WaitTaskContent(context.Background(), &clientpb.Task{
		SessionId: mock.SessionID,
		TaskId:    task.TaskId,
		Need:      0,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent(first) failed: %v", err)
	}
	if got := string(first.GetSpite().GetExecResponse().GetStdout()); got != "alpha" {
		t.Fatalf("first exec chunk = %q, want alpha", got)
	}

	second, err := rpc.WaitTaskContent(context.Background(), &clientpb.Task{
		SessionId: mock.SessionID,
		TaskId:    task.TaskId,
		Need:      1,
	})
	if err != nil {
		t.Fatalf("WaitTaskContent(second) failed: %v", err)
	}
	if got := string(second.GetSpite().GetExecResponse().GetStdout()); got != "omega" {
		t.Fatalf("second exec chunk = %q, want omega", got)
	}

	finished, err := rpc.WaitTaskFinish(context.Background(), &clientpb.Task{
		SessionId: mock.SessionID,
		TaskId:    task.TaskId,
	})
	if err != nil {
		t.Fatalf("WaitTaskFinish failed: %v", err)
	}
	if finished == nil || finished.Task == nil || finished.Spite == nil {
		t.Fatalf("WaitTaskFinish content = %#v, want populated task context", finished)
	}
	if !finished.Task.Finished {
		t.Fatalf("finished task = %#v, want finished state", finished.Task)
	}
	if finished.Task.Cur != 2 || finished.Task.Total != 2 {
		t.Fatalf("finished task progress = %d/%d, want 2/2", finished.Task.Cur, finished.Task.Total)
	}
	if got := string(finished.GetSpite().GetExecResponse().GetStdout()); got != "omega" {
		t.Fatalf("finished exec chunk = %q, want omega", got)
	}

	if errs := mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}
