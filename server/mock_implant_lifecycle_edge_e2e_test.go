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
	"github.com/chainreactors/malice-network/server/testsupport"
)

func forceSessionInactive(t *testing.T, sessionID string) (*core.Session, int64) {
	t.Helper()

	session := getRuntimeSession(t, sessionID)
	staleAt := time.Now().Add(-10 * time.Minute).Unix()
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

		return send(&implantpb.Spite{
			Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
				Stdout:     []byte("omega"),
				Pid:        7001,
				StatusCode: 0,
				End:        true,
			}},
		})
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
	if got := string(finished.GetSpite().GetExecResponse().GetStdout()); got != "omega" {
		t.Fatalf("finished exec stdout = %q, want omega", got)
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
