package tasks_test

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestTaskCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "tasks --all requests full task history",
			Argv: []string{consts.CommandTasks, "--all"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnTasks("GetTasks", func(ctx context.Context, request any) (*clientpb.Tasks, error) {
					return &clientpb.Tasks{
						Tasks: []*clientpb.Task{
							{TaskId: 9, SessionId: h.Session.SessionId, Type: consts.ModuleSleep, Cur: 1, Total: 1},
						},
					}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*clientpb.TaskRequest](t, h, "GetTasks")
				if req.SessionId != h.Session.SessionId {
					t.Fatalf("tasks session id = %q, want %q", req.SessionId, h.Session.SessionId)
				}
				if !req.All {
					t.Fatal("tasks --all should request all task history")
				}
				testsupport.RequireNoSessionEvents(t, h)
				testsupport.RequireCallee(t, md, consts.CalleeCMD)
			},
		},
		{
			Name:    "fetch_task rejects invalid ids before rpc",
			Argv:    []string{consts.CommandTaskFetch, "not-a-number"},
			WantErr: "invalid task ID",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "fetch_task forwards task lookup request",
			Argv: []string{consts.CommandTaskFetch, "7"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnTaskContexts("GetAllTaskContent", func(ctx context.Context, request any) (*clientpb.TaskContexts, error) {
					return &clientpb.TaskContexts{
						Task:    &clientpb.Task{TaskId: 7, SessionId: h.Session.SessionId, Type: consts.ModuleSleep},
						Session: testsupport.SessionClone(h.Session),
						Spites:  nil,
					}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*clientpb.Task](t, h, "GetAllTaskContent")
				if req.SessionId != h.Session.SessionId {
					t.Fatalf("fetch_task session id = %q, want %q", req.SessionId, h.Session.SessionId)
				}
				if req.TaskId != 7 {
					t.Fatalf("fetch_task id = %d, want 7", req.TaskId)
				}
				if req.Need != -1 {
					t.Fatalf("fetch_task need = %d, want -1", req.Need)
				}
				testsupport.RequireNoSessionEvents(t, h)
				testsupport.RequireCallee(t, md, consts.CalleeCMD)
			},
		},
	})
}
