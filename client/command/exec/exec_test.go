package exec_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestExecCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "run preserves executable and dashed arguments",
			Argv: []string{consts.ModuleAliasRun, "gogo.exe", "--", "-i", "127.0.0.1", "-p", "http"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ExecRequest](t, h, "Execute")
				if req.Path != "gogo.exe" {
					t.Fatalf("run path = %q, want gogo.exe", req.Path)
				}
				wantArgs := []string{"-i", "127.0.0.1", "-p", "http"}
				if len(req.Args) != len(wantArgs) {
					t.Fatalf("run args = %#v, want %#v", req.Args, wantArgs)
				}
				for i := range wantArgs {
					if req.Args[i] != wantArgs[i] {
						t.Fatalf("run args = %#v, want %#v", req.Args, wantArgs)
					}
				}
				if req.Realtime || !req.Output {
					t.Fatalf("run flags = %#v, want realtime=false output=true", req)
				}
				assertExecTaskEvent(t, h, md, consts.ModuleExecute)
			},
		},
		{
			Name: "execute disables output collection",
			Argv: []string{consts.ModuleAliasExecute, "cmd.exe", "/c", "hostname"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ExecRequest](t, h, "Execute")
				if req.Path != "cmd.exe" || len(req.Args) != 2 || req.Args[0] != "/c" || req.Args[1] != "hostname" {
					t.Fatalf("execute request = %#v", req)
				}
				if req.Realtime || req.Output {
					t.Fatalf("execute flags = %#v, want realtime=false output=false", req)
				}
				assertExecTaskEvent(t, h, md, consts.ModuleExecute)
			},
		},
		{
			Name: "shell wraps command in cmd slash-c",
			Argv: []string{consts.ModuleAliasShell, "whoami", "/all"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ExecRequest](t, h, "Execute")
				if req.Path != `C:\Windows\System32\cmd.exe` {
					t.Fatalf("shell path = %q, want cmd.exe", req.Path)
				}
				wantArgs := []string{"/c", "whoami /all"}
				if len(req.Args) != len(wantArgs) {
					t.Fatalf("shell args = %#v, want %#v", req.Args, wantArgs)
				}
				for i := range wantArgs {
					if req.Args[i] != wantArgs[i] {
						t.Fatalf("shell args = %#v, want %#v", req.Args, wantArgs)
					}
				}
				if !req.Realtime || !req.Output {
					t.Fatalf("shell flags = %#v, want realtime=true output=true", req)
				}
				assertExecTaskEvent(t, h, md, consts.ModuleExecute)
			},
		},
		{
			Name: "shell quiet disables output but keeps realtime",
			Argv: []string{consts.ModuleAliasShell, "--quiet", "dir"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ExecRequest](t, h, "Execute")
				if req.Path != `C:\Windows\System32\cmd.exe` {
					t.Fatalf("shell quiet path = %q, want cmd.exe", req.Path)
				}
				wantArgs := []string{"/c", "dir"}
				if len(req.Args) != len(wantArgs) {
					t.Fatalf("shell quiet args = %#v, want %#v", req.Args, wantArgs)
				}
				for i := range wantArgs {
					if req.Args[i] != wantArgs[i] {
						t.Fatalf("shell quiet args = %#v, want %#v", req.Args, wantArgs)
					}
				}
				if !req.Realtime || req.Output {
					t.Fatalf("shell quiet flags = %#v, want realtime=true output=false", req)
				}
				assertExecTaskEvent(t, h, md, consts.ModuleExecute)
			},
		},
		{
			Name: "powershell uses standard bypass wrapper",
			Argv: []string{consts.ModuleAliasPowershell, "Get-ChildItem", "Env:"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ExecRequest](t, h, "Execute")
				if req.Path != `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe` {
					t.Fatalf("powershell path = %q", req.Path)
				}
				wantArgs := []string{"-ExecutionPolicy", "Bypass", "-w", "hidden", "-nop", "Get-ChildItem Env:"}
				if len(req.Args) != len(wantArgs) {
					t.Fatalf("powershell args = %#v, want %#v", req.Args, wantArgs)
				}
				for i := range wantArgs {
					if req.Args[i] != wantArgs[i] {
						t.Fatalf("powershell args = %#v, want %#v", req.Args, wantArgs)
					}
				}
				if !req.Realtime || !req.Output {
					t.Fatalf("powershell flags = %#v, want realtime=true output=true", req)
				}
				assertExecTaskEvent(t, h, md, consts.ModuleExecute)
			},
		},
	})
}

func TestShellFileOutputWaitsForAsyncTask(t *testing.T) {
	h := testsupport.NewHarness(t)
	outputPath := filepath.Join(t.TempDir(), "shell-output.txt")

	waitCalled := false
	h.Recorder.OnTaskContext("WaitTaskFinish", func(_ context.Context, request any) (*clientpb.TaskContext, error) {
		waitCalled = true
		task, ok := request.(*clientpb.Task)
		if !ok {
			return nil, fmt.Errorf("wait request type = %T, want *clientpb.Task", request)
		}
		return &clientpb.TaskContext{
			Task: &clientpb.Task{
				TaskId:    task.GetTaskId(),
				SessionId: task.GetSessionId(),
				Type:      consts.ModuleExecute,
				Cur:       2,
				Total:     2,
				Finished:  true,
			},
			Session: h.Session.Session,
			Spite: &implantpb.Spite{
				Error: 6,
			},
		}, nil
	})
	h.Recorder.OnTaskContexts("GetAllTaskContent", func(_ context.Context, request any) (*clientpb.TaskContexts, error) {
		if !waitCalled {
			return nil, errors.New("GetAllTaskContent called before WaitTaskFinish")
		}
		task, ok := request.(*clientpb.Task)
		if !ok {
			return nil, fmt.Errorf("content request type = %T, want *clientpb.Task", request)
		}
		return &clientpb.TaskContexts{
			Task: &clientpb.Task{
				TaskId:    task.GetTaskId(),
				SessionId: task.GetSessionId(),
				Type:      consts.ModuleExecute,
				Cur:       2,
				Total:     2,
				Finished:  true,
			},
			Session: h.Session.Session,
			Spites: []*implantpb.Spite{
				{Error: 0},
				{Error: 6},
			},
		}, nil
	})

	if err := h.Execute(consts.ModuleAliasShell, "-f", outputPath, "whoami"); err != nil {
		t.Fatalf("execute shell with file output failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 3 {
		t.Fatalf("primary call count = %d, want 3 (Execute, WaitTaskFinish, GetAllTaskContent)", len(calls))
	}
	if calls[0].Method != "Execute" || calls[1].Method != "WaitTaskFinish" || calls[2].Method != "GetAllTaskContent" {
		t.Fatalf("primary call order = %#v, want [Execute WaitTaskFinish GetAllTaskContent]", []string{calls[0].Method, calls[1].Method, calls[2].Method})
	}

	taskReq, ok := calls[1].Request.(*clientpb.Task)
	if !ok {
		t.Fatalf("wait request type = %T, want *clientpb.Task", calls[1].Request)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file failed: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, fmt.Sprintf("task: %d true", taskReq.GetTaskId())) {
		t.Fatalf("output file = %q, want first aggregated chunk", text)
	}
	if !strings.Contains(text, fmt.Sprintf("task: %d false", taskReq.GetTaskId())) {
		t.Fatalf("output file = %q, want second aggregated chunk", text)
	}
}

func assertExecTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("exec session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
