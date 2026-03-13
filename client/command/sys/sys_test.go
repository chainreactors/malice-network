package sys_test

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestSysCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "whoami sends whoami request",
			Argv: []string{consts.ModuleWhoami},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Whoami")
				if req.Name != consts.ModuleWhoami {
					t.Fatalf("whoami request name = %q, want %q", req.Name, consts.ModuleWhoami)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleWhoami)
			},
		},
		{
			Name: "kill sends pid as input",
			Argv: []string{consts.ModuleKill, "1337"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Kill")
				if req.Name != consts.ModuleKill || req.Input != "1337" {
					t.Fatalf("kill request = %#v, want name kill and input 1337", req)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleKill)
			},
		},
		{
			Name: "ps sends process list request",
			Argv: []string{consts.ModulePs},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Ps")
				if req.Name != consts.ModulePs {
					t.Fatalf("ps request name = %q, want %q", req.Name, consts.ModulePs)
				}
				assertSysTaskEvent(t, h, md, consts.ModulePs)
			},
		},
		{
			Name: "env lists environment variables",
			Argv: []string{consts.ModuleEnv},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Env")
				if req.Name != consts.ModuleEnv {
					t.Fatalf("env request name = %q, want %q", req.Name, consts.ModuleEnv)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleEnv)
			},
		},
		{
			Name: "setenv forwards key and value",
			Argv: []string{consts.ModuleEnv, "set", "TMP", `C:\Temp`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "SetEnv")
				if req.Name != consts.ModuleSetEnv || len(req.Args) != 2 || req.Args[0] != "TMP" || req.Args[1] != `C:\Temp` {
					t.Fatalf("setenv request = %#v", req)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleSetEnv)
			},
		},
		{
			Name: "unsetenv forwards key",
			Argv: []string{consts.ModuleEnv, "unset", "TMP"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "UnsetEnv")
				if req.Name != consts.ModuleUnsetEnv || req.Input != "TMP" {
					t.Fatalf("unsetenv request = %#v", req)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleUnsetEnv)
			},
		},
		{
			Name: "netstat sends netstat request",
			Argv: []string{consts.ModuleNetstat},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Netstat")
				if req.Name != consts.ModuleNetstat {
					t.Fatalf("netstat request name = %q, want %q", req.Name, consts.ModuleNetstat)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleNetstat)
			},
		},
		{
			Name: "sysinfo sends info request",
			Argv: []string{consts.ModuleSysInfo},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Info")
				if req.Name != consts.ModuleSysInfo {
					t.Fatalf("sysinfo request name = %q, want %q", req.Name, consts.ModuleSysInfo)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleSysInfo)
			},
		},
		{
			Name: "bypass maps amsi and etw flags",
			Argv: []string{consts.ModuleBypass, "--amsi", "--etw"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.BypassRequest](t, h, "Bypass")
				if !req.AMSI || !req.ETW || req.BlockDll {
					t.Fatalf("bypass request = %#v, want AMSI/ETW true and BlockDll false", req)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleBypass)
			},
		},
		{
			Name: "wmi_query forwards namespace and args",
			Argv: []string{consts.ModuleWmiQuery, "--namespace", `root\cimv2`, "--args", "SELECT * FROM Win32_Process"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.WmiQueryRequest](t, h, "WmiQuery")
				if req.Namespace != `root\cimv2` || len(req.Args) != 1 || req.Args[0] != "SELECT * FROM Win32_Process" {
					t.Fatalf("wmi query request = %#v", req)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleWmiQuery)
			},
		},
		{
			Name: "wmi_execute parses params into key-value map",
			Argv: []string{
				consts.ModuleWmiExec,
				"--namespace", `root\cimv2`,
				"--class_name", "Win32_Process",
				"--method_name", "Create",
				"--params", "CommandLine=cmd /c calc",
				"--params", `CurrentDirectory=C:\Temp`,
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.WmiMethodRequest](t, h, "WmiExecute")
				if req.Namespace != `root\cimv2` || req.ClassName != "Win32_Process" || req.MethodName != "Create" {
					t.Fatalf("wmi execute request = %#v", req)
				}
				if len(req.Params) != 2 || req.Params["CommandLine"] != "cmd /c calc" || req.Params["CurrentDirectory"] != `C:\Temp` {
					t.Fatalf("wmi execute params = %#v", req.Params)
				}
				assertSysTaskEvent(t, h, md, consts.ModuleWmiExec)
			},
		},
		{
			Name:    "wmi_execute rejects malformed params",
			Argv:    []string{consts.ModuleWmiExec, "--params", "broken"},
			WantErr: "invalid --params value",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
	})
}

func assertSysTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("sys session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
