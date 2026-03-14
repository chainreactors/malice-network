package addon_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestLoadAddonInfersModuleFromFileExtension(t *testing.T) {
	h := testsupport.NewHarness(t)
	path := filepath.Join(t.TempDir(), "demo.dll")
	if err := os.WriteFile(path, []byte("addon-binary"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := h.Execute(consts.ModuleLoadAddon, path); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	req, md := testsupport.MustSingleCall[*implantpb.LoadAddon](t, h, "LoadAddon")
	if req.Name != "demo.dll" {
		t.Fatalf("addon name = %q, want demo.dll", req.Name)
	}
	if req.Depend != consts.ModuleExecuteDll {
		t.Fatalf("addon depend = %q, want %q", req.Depend, consts.ModuleExecuteDll)
	}
	if string(req.Bin) != "addon-binary" {
		t.Fatalf("addon binary = %q, want addon-binary", req.Bin)
	}
	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)
	assertSingleTaskEvent(t, h, consts.ModuleLoadAddon)
}

func TestLoadAddonExplicitModuleOverridesExtensionInference(t *testing.T) {
	h := testsupport.NewHarness(t)
	path := filepath.Join(t.TempDir(), "demo.dll")
	if err := os.WriteFile(path, []byte("addon-binary"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := h.Execute(consts.ModuleLoadAddon, "--module", consts.ModuleExecuteExe, path); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*implantpb.LoadAddon](t, h, "LoadAddon")
	if req.Depend != consts.ModuleExecuteExe {
		t.Fatalf("addon depend = %q, want %q", req.Depend, consts.ModuleExecuteExe)
	}
}

func TestExecuteAddonRequiresLoadedDependencyModule(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Session.Session.Addons = []*implantpb.Addon{{
		Name:   "demo",
		Depend: consts.ModuleExecuteDll,
	}}

	if err := h.Execute(consts.ModuleExecuteAddon, "demo"); err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}

	testsupport.RequireNoPrimaryCalls(t, h)
	testsupport.RequireNoSessionEvents(t, h)
}

func TestExecuteAddonForwardsSacrificeAndExecutionArgs(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Session.Session.Modules = append(h.Session.Session.Modules, consts.ModuleExecuteDll)
	h.Session.Session.Addons = []*implantpb.Addon{{
		Name:   "demo",
		Depend: consts.ModuleExecuteDll,
	}}

	err := h.Execute(
		consts.ModuleExecuteAddon,
		"--ppid", "42",
		"--argue", "notepad.exe",
		"--process", `C:\\Windows\\System32\\rundll32.exe`,
		"demo",
		"arg1",
		"arg2",
	)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	req, md := testsupport.MustSingleCall[*implantpb.ExecuteAddon](t, h, "ExecuteAddon")
	if req.Addon != "demo" {
		t.Fatalf("addon name = %q, want demo", req.Addon)
	}
	if req.ExecuteBinary == nil {
		t.Fatal("execute binary is nil")
	}
	if len(req.ExecuteBinary.Args) != 2 || req.ExecuteBinary.Args[0] != "arg1" || req.ExecuteBinary.Args[1] != "arg2" {
		t.Fatalf("execute args = %v, want [arg1 arg2]", req.ExecuteBinary.Args)
	}
	if req.ExecuteBinary.ProcessName != `C:\\Windows\\System32\\rundll32.exe` {
		t.Fatalf("process name = %q", req.ExecuteBinary.ProcessName)
	}
	if req.ExecuteBinary.Sacrifice == nil {
		t.Fatal("sacrifice config should be set for DLL addon execution")
	}
	if req.ExecuteBinary.Sacrifice.Ppid != 42 {
		t.Fatalf("sacrifice ppid = %d, want 42", req.ExecuteBinary.Sacrifice.Ppid)
	}
	if req.ExecuteBinary.Sacrifice.Argue != "notepad.exe" {
		t.Fatalf("sacrifice argue = %q, want notepad.exe", req.ExecuteBinary.Sacrifice.Argue)
	}
	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)
	assertSingleTaskEvent(t, h, consts.ModuleExecuteAddon)
}

func TestExecuteAddonUsesDefaultCommandProcessAndSessionArch(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Session.Session.Modules = append(h.Session.Session.Modules, consts.ModuleExecuteDll)
	h.Session.Session.Addons = []*implantpb.Addon{{
		Name:   "demo",
		Depend: consts.ModuleExecuteDll,
	}}

	if err := h.Execute(consts.ModuleExecuteAddon, "demo"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*implantpb.ExecuteAddon](t, h, "ExecuteAddon")
	if req.ExecuteBinary == nil {
		t.Fatal("execute binary is nil")
	}
	if req.ExecuteBinary.ProcessName != `C:\\Windows\\System32\\svchost.exe` {
		t.Fatalf("process name = %q, want default execute flag process", req.ExecuteBinary.ProcessName)
	}
	if req.ExecuteBinary.Arch != consts.MapArch(h.Session.Os.Arch) {
		t.Fatalf("arch = %d, want %d", req.ExecuteBinary.Arch, consts.MapArch(h.Session.Os.Arch))
	}
}

func TestExecuteAddonRpcFailureDoesNotEmitSessionEvent(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Session.Session.Modules = append(h.Session.Session.Modules, consts.ModuleExecuteDll)
	h.Session.Session.Addons = []*implantpb.Addon{{
		Name:   "demo",
		Depend: consts.ModuleExecuteDll,
	}}
	h.Recorder.OnTask("ExecuteAddon", func(ctx context.Context, request any) (*clientpb.Task, error) {
		return nil, context.DeadlineExceeded
	})

	if err := h.Execute(consts.ModuleExecuteAddon, "demo"); err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "ExecuteAddon" {
		t.Fatalf("calls = %#v, want single ExecuteAddon call", calls)
	}
	testsupport.RequireNoSessionEvents(t, h)
}

func assertSingleTaskEvent(t testing.TB, h *testsupport.Harness, wantType string) {
	t.Helper()

	event, md := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil {
		t.Fatal("session event task is nil")
	}
	if event.Task.Type != wantType {
		t.Fatalf("event task type = %q, want %q", event.Task.Type, wantType)
	}
	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)
}
