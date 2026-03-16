package tasks_test

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
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
			Name: "list_task sends implant task list request",
			Argv: []string{consts.ModuleListTask},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "ListTasks")
				if req.Name != consts.ModuleListTask {
					t.Fatalf("list_task name = %q, want %q", req.Name, consts.ModuleListTask)
				}
				assertTaskEvent(t, h, md, consts.ModuleListTask)
			},
		},
		{
			Name: "query_task forwards task control request",
			Argv: []string{consts.ModuleQueryTask, "7"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.TaskCtrl](t, h, "QueryTask")
				if req.TaskId != 7 {
					t.Fatalf("query task id = %d, want 7", req.TaskId)
				}
				if req.Op != consts.ModuleQueryTask {
					t.Fatalf("query task op = %q, want %q", req.Op, consts.ModuleQueryTask)
				}
				assertTaskEvent(t, h, md, consts.ModuleQueryTask)
			},
		},
		{
			Name:    "query_task rejects invalid ids before rpc",
			Argv:    []string{consts.ModuleQueryTask, "not-a-number"},
			WantErr: "invalid syntax",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "cancel_task forwards task control request",
			Argv: []string{consts.ModuleCancelTask, "7"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.TaskCtrl](t, h, "CancelTask")
				if req.TaskId != 7 {
					t.Fatalf("cancel task id = %d, want 7", req.TaskId)
				}
				if req.Op != consts.ModuleCancelTask {
					t.Fatalf("cancel task op = %q, want %q", req.Op, consts.ModuleCancelTask)
				}
				assertTaskEvent(t, h, md, consts.ModuleCancelTask)
			},
		},
		{
			Name:    "cancel_task rejects invalid ids before rpc",
			Argv:    []string{consts.ModuleCancelTask, "not-a-number"},
			WantErr: "invalid syntax",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
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

func assertTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Op != consts.CtrlSessionTask {
		t.Fatalf("session event op = %q, want %q", event.Op, consts.CtrlSessionTask)
	}
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
