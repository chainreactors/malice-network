package exec_test

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
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
				if len(req.Args) != 2 || req.Args[0] != "/c" || req.Args[1] != "whoami /all" {
					t.Fatalf("shell args = %#v, want [/c \"whoami /all\"]", req.Args)
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
				if req.Path != `C:\Windows\System32\cmd.exe` || len(req.Args) != 2 || req.Args[1] != "dir" {
					t.Fatalf("shell quiet request = %#v", req)
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
