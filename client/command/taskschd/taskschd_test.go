package taskschd_test

import (
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestTaskSchdCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "list sends task schedule list request",
			Argv: []string{consts.CommandTaskSchd, "list"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "TaskSchdList")
				if req.Name != consts.ModuleTaskSchdList {
					t.Fatalf("taskschd list name = %q, want %q", req.Name, consts.ModuleTaskSchdList)
				}
				assertTaskSchdEvent(t, h, md, consts.ModuleTaskSchdList)
			},
		},
		{
			Name: "create maps task configuration",
			Argv: []string{
				consts.CommandTaskSchd, "create",
				"--name", "Cleanup",
				"--path", `C:\Windows\cleanup.exe`,
				"--task_folder", `\Ops`,
				"--trigger_type", "startup",
				"--start_boundary", "2026-03-14T09:00:00",
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.TaskScheduleRequest](t, h, "TaskSchdCreate")
				if req.Type != consts.ModuleTaskSchdCreate || req.Taskschd == nil {
					t.Fatalf("taskschd create request = %#v", req)
				}
				if req.Taskschd.Name != "Cleanup" || req.Taskschd.Path != `\Ops` || req.Taskschd.ExecutablePath != `C:\Windows\cleanup.exe` {
					t.Fatalf("taskschd create payload = %#v", req.Taskschd)
				}
				if req.Taskschd.TriggerType != 8 {
					t.Fatalf("trigger type = %d, want 8", req.Taskschd.TriggerType)
				}
				if req.Taskschd.StartBoundary != "2026-03-14T09:00:00" {
					t.Fatalf("start boundary = %q, want 2026-03-14T09:00:00", req.Taskschd.StartBoundary)
				}
				assertTaskSchdEvent(t, h, md, consts.ModuleTaskSchdCreate)
			},
		},
		{
			Name:    "create enforces required flags",
			Argv:    []string{consts.CommandTaskSchd, "create", "--task_folder", `\Ops`},
			WantErr: "required flag(s)",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				if err == nil || !strings.Contains(err.Error(), "name") || !strings.Contains(err.Error(), "path") {
					t.Fatalf("taskschd create error = %v, want required name and path flags", err)
				}
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "start forwards name and folder",
			Argv: []string{consts.CommandTaskSchd, "start", "Cleanup", "--task_folder", `\Ops`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.TaskScheduleRequest](t, h, "TaskSchdStart")
				if req.Type != consts.ModuleTaskSchdStart || req.Taskschd == nil || req.Taskschd.Name != "Cleanup" || req.Taskschd.Path != `\Ops` {
					t.Fatalf("taskschd start request = %#v", req)
				}
				assertTaskSchdEvent(t, h, md, consts.ModuleTaskSchdStart)
			},
		},
		{
			Name: "stop forwards name and folder",
			Argv: []string{consts.CommandTaskSchd, "stop", "Cleanup", "--task_folder", `\Ops`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.TaskScheduleRequest](t, h, "TaskSchdStop")
				if req.Type != consts.ModuleTaskSchdStop || req.Taskschd == nil || req.Taskschd.Name != "Cleanup" || req.Taskschd.Path != `\Ops` {
					t.Fatalf("taskschd stop request = %#v", req)
				}
				assertTaskSchdEvent(t, h, md, consts.ModuleTaskSchdStop)
			},
		},
		{
			Name: "delete forwards name and folder",
			Argv: []string{consts.CommandTaskSchd, "delete", "Cleanup", "--task_folder", `\Ops`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.TaskScheduleRequest](t, h, "TaskSchdDelete")
				if req.Type != consts.ModuleTaskSchdDelete || req.Taskschd == nil || req.Taskschd.Name != "Cleanup" || req.Taskschd.Path != `\Ops` {
					t.Fatalf("taskschd delete request = %#v", req)
				}
				assertTaskSchdEvent(t, h, md, consts.ModuleTaskSchdDelete)
			},
		},
		{
			Name: "query forwards name and folder",
			Argv: []string{consts.CommandTaskSchd, "query", "Cleanup", "--task_folder", `\Ops`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.TaskScheduleRequest](t, h, "TaskSchdQuery")
				if req.Type != consts.ModuleTaskSchdQuery || req.Taskschd == nil || req.Taskschd.Name != "Cleanup" || req.Taskschd.Path != `\Ops` {
					t.Fatalf("taskschd query request = %#v", req)
				}
				assertTaskSchdEvent(t, h, md, consts.ModuleTaskSchdQuery)
			},
		},
		{
			Name: "run forwards name and folder",
			Argv: []string{consts.CommandTaskSchd, "run", "Cleanup", "--task_folder", `\Ops`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.TaskScheduleRequest](t, h, "TaskSchdRun")
				if req.Type != consts.ModuleTaskSchdRun || req.Taskschd == nil || req.Taskschd.Name != "Cleanup" || req.Taskschd.Path != `\Ops` {
					t.Fatalf("taskschd run request = %#v", req)
				}
				assertTaskSchdEvent(t, h, md, consts.ModuleTaskSchdRun)
			},
		},
	})
}

func TestTaskSchdCreateTriggerAliases(t *testing.T) {
	cases := []struct {
		name    string
		trigger string
		want    uint32
	}{
		{name: "daily alias", trigger: "daily", want: 2},
		{name: "weekly alias", trigger: "weekly", want: 3},
		{name: "monthly alias", trigger: "monthly", want: 4},
		{name: "atlogon alias", trigger: "atlogon", want: 9},
		{name: "startup alias", trigger: "startup", want: 8},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h := testsupport.NewHarness(t)
			err := h.Execute(
				consts.CommandTaskSchd, "create",
				"--name", "AliasTask",
				"--path", `C:\Windows\alias.exe`,
				"--trigger_type", tc.trigger,
			)
			if err != nil {
				t.Fatalf("execute failed: %v", err)
			}

			req, md := testsupport.MustSingleCall[*implantpb.TaskScheduleRequest](t, h, "TaskSchdCreate")
			if req.Taskschd == nil || req.Taskschd.TriggerType != tc.want {
				t.Fatalf("trigger type for %q = %#v, want %d", tc.trigger, req.Taskschd, tc.want)
			}
			assertTaskSchdEvent(t, h, md, consts.ModuleTaskSchdCreate)
		})
	}
}

func assertTaskSchdEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("taskschd session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
