package addon_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	commandpkg "github.com/chainreactors/malice-network/client/command"
	addoncmd "github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/spf13/cobra"
)

func TestListAddonSendsModuleRequest(t *testing.T) {
	h := testsupport.NewHarness(t)

	if err := h.Execute(consts.ModuleListAddon); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "ListAddon")
	if req.Name != consts.ModuleListAddon {
		t.Fatalf("addon list name = %q, want %q", req.Name, consts.ModuleListAddon)
	}
	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)
	assertSingleTaskEvent(t, h, consts.ModuleListAddon)
}

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

func TestRefreshAddonCommandReplacesDynamicMenuCommands(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Session.Session.Modules = append(h.Session.Session.Modules, consts.ModuleExecute)
	root := commandpkg.BindImplantCommands(h.Console)()
	root.SilenceErrors = true
	root.SilenceUsage = true
	h.Console.App.Menu(consts.ImplantMenu).Command = root
	h.Console.App.SwitchMenu(consts.ImplantMenu)

	initial := []*implantpb.Addon{
		{Name: "demo-a", Type: "assembly", Depend: consts.ModuleExecute},
		{Name: "demo-b", Type: "bof", Depend: consts.ModuleExecute},
	}
	h.Session.Session.Addons = initial
	if err := addoncmd.RefreshAddonCommand(initial, h.Console); err != nil {
		t.Fatalf("RefreshAddonCommand(initial) failed: %v", err)
	}
	if !hasCommand(root, "demo-a") || !hasCommand(root, "demo-b") {
		t.Fatalf("addon commands after initial refresh = %v, want demo-a and demo-b", commandNames(root))
	}

	updated := []*implantpb.Addon{
		{Name: "demo-c", Type: "assembly", Depend: consts.ModuleExecute},
	}
	h.Session.Session.Addons = updated
	if err := addoncmd.RefreshAddonCommand(updated, h.Console); err != nil {
		t.Fatalf("RefreshAddonCommand(updated) failed: %v", err)
	}
	if hasCommand(root, "demo-a") || hasCommand(root, "demo-b") || !hasCommand(root, "demo-c") {
		t.Fatalf("addon commands after replacement = %v, want only demo-c among dynamic addon commands", commandNames(root))
	}

	cmd := findCommand(root, "demo-c")
	if cmd == nil {
		t.Fatal("demo-c command not found after refresh")
	}
	if cmd.GroupID != consts.AddonGroup {
		t.Fatalf("demo-c group = %q, want %q", cmd.GroupID, consts.AddonGroup)
	}

	h.Console.App.Shell().Line().Set([]rune("demo-c arg1 arg2")...)
	root.SetArgs([]string{"--use", h.Session.SessionId, "demo-c", "arg1", "arg2"})
	if err := root.Execute(); err != nil {
		t.Fatalf("dynamic addon command execute failed: %v", err)
	}

	req, md := testsupport.MustSingleCall[*implantpb.ExecuteAddon](t, h, "ExecuteAddon")
	if req.Addon != "demo-c" {
		t.Fatalf("execute addon name = %q, want demo-c", req.Addon)
	}
	if req.ExecuteBinary == nil {
		t.Fatal("execute binary is nil")
	}
	if len(req.ExecuteBinary.Args) != 2 || req.ExecuteBinary.Args[0] != "arg1" || req.ExecuteBinary.Args[1] != "arg2" {
		t.Fatalf("execute args = %v, want [arg1 arg2]", req.ExecuteBinary.Args)
	}
	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)
	assertSingleTaskEvent(t, h, consts.ModuleExecuteAddon)
}

func TestLoadAddonFinishCallbackRefreshesCommandsFromUpdatedSession(t *testing.T) {
	h := testsupport.NewHarness(t)
	root := commandpkg.BindImplantCommands(h.Console)()
	root.SilenceErrors = true
	root.SilenceUsage = true
	h.Console.App.Menu(consts.ImplantMenu).Command = root
	h.Console.App.SwitchMenu(consts.ImplantMenu)

	addonName := "demo-addon"
	taskID := uint32(77)
	path := filepath.Join(t.TempDir(), "demo.exe")
	if err := os.WriteFile(path, []byte("addon-binary"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	h.Recorder.OnTask("LoadAddon", func(ctx context.Context, request any) (*clientpb.Task, error) {
		return &clientpb.Task{
			TaskId:    taskID,
			SessionId: h.Session.SessionId,
			Type:      consts.ModuleLoadAddon,
			Cur:       1,
			Total:     1,
		}, nil
	})

	updated := testsupport.SessionClone(h.Session)
	updated.Addons = []*implantpb.Addon{{
		Name:   addonName,
		Type:   "exe",
		Depend: consts.ModuleExecuteExe,
	}}
	h.SetSessionResponse(updated)

	if err := h.Execute(consts.ModuleLoadAddon, "--name", addonName, "--module", consts.ModuleExecuteExe, path); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	callbackID := fmt.Sprintf("%s-%d", h.Session.SessionId, taskID)
	rawCallback, ok := h.Console.FinishCallbacks.Load(callbackID)
	if !ok {
		t.Fatalf("finish callback %q not registered", callbackID)
	}

	callback, ok := rawCallback.(iomclient.TaskCallback)
	if !ok {
		t.Fatalf("finish callback type = %T, want iomclient.TaskCallback", rawCallback)
	}
	callback(&clientpb.TaskContext{
		Task: &clientpb.Task{
			TaskId:    taskID,
			SessionId: h.Session.SessionId,
			Type:      consts.ModuleLoadAddon,
			Finished:  true,
		},
		Session: updated,
	})

	if !hasCommand(root, addonName) {
		t.Fatalf("dynamic addon commands = %v, want %q", commandNames(root), addonName)
	}
}

func assertSingleTaskEvent(t testing.TB, h *testsupport.Harness, wantType string) {
	t.Helper()

	event, md := testsupport.MustSingleSessionEvent(t, h)
	if event.Op != consts.CtrlSessionTask {
		t.Fatalf("session event op = %q, want %q", event.Op, consts.CtrlSessionTask)
	}
	if event.Task == nil {
		t.Fatal("session event task is nil")
	}
	if event.Task.Type != wantType {
		t.Fatalf("event task type = %q, want %q", event.Task.Type, wantType)
	}
	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)
}

func hasCommand(root interface{ Commands() []*cobra.Command }, name string) bool {
	return findCommand(root, name) != nil
}

func findCommand(root interface{ Commands() []*cobra.Command }, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func commandNames(root interface{ Commands() []*cobra.Command }) []string {
	commands := root.Commands()
	names := make([]string, 0, len(commands))
	for _, cmd := range commands {
		names = append(names, cmd.Name())
	}
	return names
}
