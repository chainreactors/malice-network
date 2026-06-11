//go:build realimplant

package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	serverrpc "github.com/chainreactors/malice-network/server/rpc"
	"github.com/chainreactors/malice-network/server/testsupport"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

type realCommandFixture struct {
	h       *testsupport.ControlPlaneHarness
	implant *testsupport.RealImplant
	client  *testsupport.ClientHarness
}

type discoveredService struct {
	Name        string
	DisplayName string
}

type discoveredSchedule struct {
	Name string
	Path string
}

func newRealCommandFixture(t *testing.T) *realCommandFixture {
	t.Helper()

	testsupport.RequireRealImplantEnv(t)

	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("real-command-listener-%d", time.Now().UnixNano())
	pipelineName := fmt.Sprintf("real-command-pipe-%d", time.Now().UnixNano())
	implant := testsupport.NewRealImplant(t, h, testsupport.NewRealTCPPipeline(t, listenerName, pipelineName))
	if err := implant.Start(t); err != nil {
		t.Fatalf("real implant start failed: %v", err)
	}

	clientHarness := testsupport.NewClientHarness(t, h)
	clientHarness.Console.NewConsole()

	initialCheckin := mustStoredSession(t, h, implant.SessionID).GetLastCheckin()
	testsupport.WaitForCondition(t, 10*time.Second, func() bool {
		return clientHarness.Console.Sessions[implant.SessionID] != nil
	}, "client session cache to include real implant")
	testsupport.WaitForCondition(t, 12*time.Second, func() bool {
		session, err := h.GetSession(implant.SessionID)
		return err == nil && session.GetLastCheckin() > initialCheckin
	}, "real implant post-register checkin")

	clientSession := mustClientSession(t, clientHarness.Console, implant.SessionID)
	requireModulePresent(t, clientSession.Modules, consts.ModulePwd)
	requireModulePresent(t, clientSession.Modules, consts.ModuleLs)
	requireModulePresent(t, clientSession.Modules, consts.ModuleSysInfo)
	requireModulePresent(t, clientSession.Modules, consts.ModuleExecute)

	return &realCommandFixture{
		h:       h,
		implant: implant,
		client:  clientHarness,
	}
}

func mustClientSession(t testing.TB, con interface {
	GetOrUpdateSession(string) (*iomclient.Session, error)
}, sessionID string) *iomclient.Session {
	t.Helper()

	session, err := con.GetOrUpdateSession(sessionID)
	if err != nil {
		t.Fatalf("GetOrUpdateSession(%q) failed: %v", sessionID, err)
	}
	if session == nil {
		t.Fatalf("GetOrUpdateSession(%q) returned nil", sessionID)
	}
	return session
}

func mustStoredSession(t testing.TB, h *testsupport.ControlPlaneHarness, sessionID string) *clientpb.Session {
	t.Helper()

	session, err := h.GetSession(sessionID)
	if err != nil {
		t.Fatalf("GetSession(%q) failed: %v", sessionID, err)
	}
	if session == nil {
		t.Fatalf("GetSession(%q) returned nil", sessionID)
	}
	return session
}

func mustRuntimeSession(t testing.TB, h *testsupport.ControlPlaneHarness, sessionID string) *clientpb.Session {
	t.Helper()

	session, err := h.GetRuntimeSession(sessionID)
	if err != nil {
		t.Fatalf("GetRuntimeSession(%q) failed: %v", sessionID, err)
	}
	if session == nil {
		t.Fatalf("GetRuntimeSession(%q) returned nil", sessionID)
	}
	return session
}

func waitCommandTaskFinish(t testing.TB, con *testsupport.ClientHarness, task *clientpb.Task) *clientpb.TaskContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	content, err := con.Console.Rpc.WaitTaskFinish(ctx, &clientpb.Task{
		SessionId: task.GetSessionId(),
		TaskId:    task.GetTaskId(),
	})
	if err != nil {
		t.Fatalf("WaitTaskFinish(%s-%d) failed: %v", task.GetSessionId(), task.GetTaskId(), err)
	}
	if content == nil || content.GetTask() == nil || content.GetSpite() == nil {
		t.Fatalf("WaitTaskFinish(%s-%d) returned incomplete content: %#v", task.GetSessionId(), task.GetTaskId(), content)
	}
	return content
}

func getAllTaskContent(t testing.TB, con *testsupport.ClientHarness, task *clientpb.Task) *clientpb.TaskContexts {
	t.Helper()

	content, err := con.Console.Rpc.GetAllTaskContent(context.Background(), &clientpb.Task{
		SessionId: task.GetSessionId(),
		TaskId:    task.GetTaskId(),
	})
	if err != nil {
		t.Fatalf("GetAllTaskContent(%s-%d) failed: %v", task.GetSessionId(), task.GetTaskId(), err)
	}
	if content == nil || content.GetTask() == nil {
		t.Fatalf("GetAllTaskContent(%s-%d) returned incomplete content: %#v", task.GetSessionId(), task.GetTaskId(), content)
	}
	return content
}

func waitCommandTaskContent(t testing.TB, con *testsupport.ClientHarness, task *clientpb.Task, need int32, timeout time.Duration) (*clientpb.TaskContext, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return con.Console.Rpc.WaitTaskContent(ctx, &clientpb.Task{
		SessionId: task.GetSessionId(),
		TaskId:    task.GetTaskId(),
		Need:      need,
	})
}

func (f *realCommandFixture) executeWait(t testing.TB, wantType string, argv ...string) *clientpb.Task {
	t.Helper()

	task, err := f.executeMaybeWait(t, argv...)
	if err != nil {
		t.Fatalf("execute %q failed: %v", strings.Join(argv, " "), err)
	}
	if task == nil {
		t.Fatalf("session %s last task is nil after %q", f.implant.SessionID, strings.Join(argv, " "))
	}
	if wantType != "" && task.GetType() != wantType {
		t.Fatalf("task type = %q, want %q", task.GetType(), wantType)
	}
	return task
}

func (f *realCommandFixture) executeNoWait(t testing.TB, wantType string, argv ...string) *clientpb.Task {
	t.Helper()

	task, err := f.executeCommand(t, false, argv...)
	if err != nil {
		t.Fatalf("execute %q failed: %v", strings.Join(argv, " "), err)
	}
	if task == nil {
		t.Fatalf("session %s last task is nil after %q", f.implant.SessionID, strings.Join(argv, " "))
	}
	if wantType != "" && task.GetType() != wantType {
		t.Fatalf("task type = %q, want %q", task.GetType(), wantType)
	}
	return task
}

func (f *realCommandFixture) executeMaybeWait(t testing.TB, argv ...string) (*clientpb.Task, error) {
	t.Helper()

	return f.executeCommand(t, true, argv...)
}

func (f *realCommandFixture) executeCommand(t testing.TB, wait bool, argv ...string) (*clientpb.Task, error) {
	t.Helper()

	root := f.client.Console.App.Menu(consts.ImplantMenu).Command
	if root == nil || root.Flags().Lookup("use") == nil {
		root = ImplantCmd(f.client.Console)
		root.SilenceErrors = true
		root.SilenceUsage = true
		f.client.Console.App.Menu(consts.ImplantMenu).Command = root
	}

	RegisterImplantFunc(f.client.Console)

	args := []string{"--use", f.implant.SessionID}
	if wait {
		args = append(args, "--wait")
	}
	args = append(args, argv...)
	f.client.Console.App.Shell().Line().Set([]rune(strings.Join(append([]string{"implant"}, args...), " "))...)
	root.SetArgs(args)
	err := root.Execute()

	session := mustClientSession(t, f.client.Console, f.implant.SessionID)
	if session.LastTask == nil {
		return nil, err
	}
	return proto.Clone(session.LastTask).(*clientpb.Task), err
}

func findTaskInfo(tasks []*implantpb.TaskInfo, taskID uint32) *implantpb.TaskInfo {
	for _, task := range tasks {
		if task != nil && task.GetTaskId() == taskID {
			return task
		}
	}
	return nil
}

func findAddon(addons []*implantpb.Addon, name string) *implantpb.Addon {
	for _, addon := range addons {
		if addon != nil && addon.GetName() == name {
			return addon
		}
	}
	return nil
}

func implantMenuHasCommand(root *cobra.Command, name string) bool {
	if root == nil {
		return false
	}
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return true
		}
	}
	return false
}

func waitForClientAddon(t testing.TB, f *realCommandFixture, name string) *implantpb.Addon {
	t.Helper()

	var found *implantpb.Addon
	testsupport.WaitForCondition(t, 10*time.Second, func() bool {
		session := mustClientSession(t, f.client.Console, f.implant.SessionID)
		found = findAddon(session.GetAddons(), name)
		return found != nil
	}, "client addon cache to include "+name)
	return found
}

func waitForStoredAddon(t testing.TB, f *realCommandFixture, name string) *implantpb.Addon {
	t.Helper()

	var found *implantpb.Addon
	testsupport.WaitForCondition(t, 10*time.Second, func() bool {
		session := mustStoredSession(t, f.h, f.implant.SessionID)
		found = findAddon(session.GetAddons(), name)
		return found != nil
	}, "stored addon cache to include "+name)
	return found
}

func registerAndStartPipeline(t testing.TB, pipeline *clientpb.Pipeline) {
	t.Helper()

	clone := proto.Clone(pipeline).(*clientpb.Pipeline)
	if _, err := (&serverrpc.Server{}).RegisterPipeline(context.Background(), clone); err != nil {
		t.Fatalf("RegisterPipeline(%s) failed: %v", clone.GetName(), err)
	}
	if _, err := (&serverrpc.Server{}).StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       clone.GetName(),
		ListenerId: clone.GetListenerId(),
		Pipeline:   proto.Clone(clone).(*clientpb.Pipeline),
	}); err != nil {
		t.Fatalf("StartPipeline(%s) failed: %v", clone.GetName(), err)
	}
}

func stopPipeline(t testing.TB, pipeline *clientpb.Pipeline) {
	t.Helper()

	clone := proto.Clone(pipeline).(*clientpb.Pipeline)
	if _, err := (&serverrpc.Server{}).StopPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       clone.GetName(),
		ListenerId: clone.GetListenerId(),
		Pipeline:   clone,
	}); err != nil {
		t.Fatalf("StopPipeline(%s) failed: %v", clone.GetName(), err)
	}
}

func requireModulePresent(t testing.TB, modules []string, want string) {
	t.Helper()

	for _, module := range modules {
		if module == want {
			return
		}
	}
	t.Fatalf("module list %v does not contain %q", modules, want)
}

func requireSessionModules(t testing.TB, f *realCommandFixture, wants ...string) {
	t.Helper()

	session := mustClientSession(t, f.client.Console, f.implant.SessionID)
	for _, want := range wants {
		requireModulePresent(t, session.Modules, want)
	}
}

func pickService(services []*implantpb.Service) discoveredService {
	candidates := []string{"Schedule", "EventLog", "Spooler"}
	for _, want := range candidates {
		for _, service := range services {
			if service == nil || service.GetConfig() == nil {
				continue
			}
			if service.GetConfig().GetName() == want {
				return discoveredService{
					Name:        service.GetConfig().GetName(),
					DisplayName: service.GetConfig().GetDisplayName(),
				}
			}
		}
	}

	for _, service := range services {
		if service == nil || service.GetConfig() == nil {
			continue
		}
		if service.GetConfig().GetName() != "" {
			return discoveredService{
				Name:        service.GetConfig().GetName(),
				DisplayName: service.GetConfig().GetDisplayName(),
			}
		}
	}

	return discoveredService{}
}

func pickSchedule(schedules []*implantpb.TaskSchedule) discoveredSchedule {
	for _, schedule := range schedules {
		if schedule == nil {
			continue
		}
		if schedule.GetName() != "" && schedule.GetPath() != "" {
			return discoveredSchedule{
				Name: schedule.GetName(),
				Path: schedule.GetPath(),
			}
		}
	}

	return discoveredSchedule{}
}

func deriveTaskFolder(taskPath, taskName string) string {
	if taskPath == "" {
		return `\`
	}
	if !strings.HasSuffix(taskPath, `\`+taskName) {
		return taskPath
	}

	idx := strings.LastIndex(taskPath, `\`)
	if idx <= 0 {
		return `\`
	}
	return taskPath[:idx]
}

func isElevatedSession(t testing.TB, f *realCommandFixture) bool {
	t.Helper()

	task := f.executeWait(
		t,
		consts.ModuleExecute,
		consts.ModuleAliasRun,
		"powershell.exe",
		"--",
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		"[bool](([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator))",
	)
	content := waitCommandTaskFinish(t, f.client, task)
	resp := content.GetSpite().GetExecResponse()
	if resp == nil {
		t.Fatal("elevation check exec response is nil")
	}
	return strings.Contains(strings.ToLower(string(resp.GetStdout())), "true")
}

func normalizeWindowsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.ReplaceAll(path, "/", `\`)
	path = filepath.Clean(path)
	if len(path) == 2 && strings.HasSuffix(path, ":") {
		path += `\`
	}
	return strings.ToLower(path)
}

func waitExecResponse(t testing.TB, f *realCommandFixture, task *clientpb.Task) *implantpb.ExecResponse {
	t.Helper()

	content := waitCommandTaskFinish(t, f.client, task)
	resp := content.GetSpite().GetExecResponse()
	if resp == nil {
		t.Fatalf("exec response is nil for task %d", task.GetTaskId())
	}
	return resp
}

func executeProgram(t testing.TB, f *realCommandFixture, argv ...string) *implantpb.ExecResponse {
	t.Helper()

	task := f.executeWait(t, consts.ModuleExecute, argv...)
	return waitExecResponse(t, f, task)
}

func findEnvValue(kv map[string]string, key string) (string, bool) {
	for actualKey, value := range kv {
		if strings.EqualFold(actualKey, key) {
			return value, true
		}
	}
	return "", false
}

func requireLsContains(t testing.TB, files []*implantpb.FileInfo, want string) {
	t.Helper()

	want = strings.ToLower(want)
	for _, file := range files {
		if strings.ToLower(file.GetName()) == want {
			return
		}
	}
	t.Fatalf("ls response files = %#v, want contains %q", files, want)
}

func requireLsNotContains(t testing.TB, files []*implantpb.FileInfo, want string) {
	t.Helper()

	want = strings.ToLower(want)
	for _, file := range files {
		if strings.ToLower(file.GetName()) == want {
			t.Fatalf("ls response files = %#v, want no %q", files, want)
		}
	}
}

func getTaskContextsByType(t testing.TB, f *realCommandFixture, contextType string, task *clientpb.Task) []*clientpb.Context {
	t.Helper()

	contexts, err := f.client.Console.Rpc.GetContexts(context.Background(), &clientpb.Context{
		Type: contextType,
		Task: &clientpb.Task{
			SessionId: task.GetSessionId(),
			TaskId:    task.GetTaskId(),
		},
	})
	if err != nil {
		t.Fatalf("GetContexts(%s,%s-%d) failed: %v", contextType, task.GetSessionId(), task.GetTaskId(), err)
	}
	return contexts.GetContexts()
}

func parseUint32FromOutput(output string) uint32 {
	output = strings.TrimSpace(output)
	output = strings.Trim(output, "\r\n\t ")
	var pid uint32
	fmt.Sscanf(output, "%d", &pid)
	return pid
}

func TestRealImplantCommandBasicModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	t.Run("sysinfo", func(t *testing.T) {
		task := f.executeWait(t, consts.ModuleSysInfo, consts.ModuleSysInfo)

		content := waitCommandTaskFinish(t, f.client, task)
		info := content.GetSpite().GetSysinfo()
		if info == nil {
			t.Fatal("sysinfo response is nil")
		}
		if info.GetWorkdir() == "" || info.GetFilepath() == "" {
			t.Fatalf("sysinfo workdir/filepath = %q/%q, want non-empty", info.GetWorkdir(), info.GetFilepath())
		}
		if info.GetOs() == nil || info.GetOs().GetName() == "" || info.GetProcess() == nil || info.GetProcess().GetName() == "" {
			t.Fatalf("sysinfo response = %#v, want non-empty os/process", info)
		}

		testsupport.WaitForCondition(t, 5*time.Second, func() bool {
			session := mustStoredSession(t, f.h, f.implant.SessionID)
			return session.GetWorkdir() == info.GetWorkdir() && session.GetFilepath() == info.GetFilepath()
		}, "stored sysinfo update")
		testsupport.WaitForCondition(t, 5*time.Second, func() bool {
			session := mustClientSession(t, f.client.Console, f.implant.SessionID)
			return session.GetWorkdir() == info.GetWorkdir() && session.GetFilepath() == info.GetFilepath()
		}, "client sysinfo cache update")
	})

	t.Run("pwd", func(t *testing.T) {
		task := f.executeWait(t, consts.ModulePwd, consts.ModulePwd)

		content := waitCommandTaskFinish(t, f.client, task)
		output := strings.TrimSpace(content.GetSpite().GetResponse().GetOutput())
		if output == "" {
			t.Fatal("pwd output should not be empty")
		}

		session := mustClientSession(t, f.client.Console, f.implant.SessionID)
		if normalizeWindowsPath(output) != normalizeWindowsPath(session.GetWorkdir()) {
			t.Fatalf("pwd output = %q, want client workdir %q", output, session.GetWorkdir())
		}
	})

	t.Run("ls", func(t *testing.T) {
		workdir := mustClientSession(t, f.client.Console, f.implant.SessionID).GetWorkdir()
		task := f.executeWait(t, consts.ModuleLs, consts.ModuleLs, workdir)

		content := waitCommandTaskFinish(t, f.client, task)
		response := content.GetSpite().GetLsResponse()
		if response == nil {
			t.Fatal("ls response is nil")
		}
		if !response.GetExists() {
			t.Fatalf("ls response = %#v, want exists=true", response)
		}
		if normalizeWindowsPath(response.GetPath()) != normalizeWindowsPath(workdir) {
			t.Fatalf("ls path = %q, want %q", response.GetPath(), workdir)
		}
	})

	t.Run("run", func(t *testing.T) {
		task := f.executeWait(t, consts.ModuleExecute, consts.ModuleAliasRun, "cmd.exe", "/c", "echo", "real-command-e2e")

		content := waitCommandTaskFinish(t, f.client, task)
		execResp := content.GetSpite().GetExecResponse()
		if execResp == nil {
			t.Fatal("run exec response is nil")
		}
		if execResp.GetStatusCode() != 0 {
			t.Fatalf("run status code = %d, want 0", execResp.GetStatusCode())
		}
		if !strings.Contains(strings.ToLower(string(execResp.GetStdout())), "real-command-e2e") {
			t.Fatalf("run stdout = %q, want real-command-e2e", string(execResp.GetStdout()))
		}

		allContent := getAllTaskContent(t, f.client, task)
		if len(allContent.GetSpites()) == 0 {
			t.Fatal("run task should persist at least one task content entry")
		}
	})

	t.Run("sleep", func(t *testing.T) {
		task := f.executeWait(t, consts.ModuleSleep, consts.ModuleSleep, "7", "--jitter", "0.15")

		content := waitCommandTaskFinish(t, f.client, task)
		if content.GetTask().GetFinished() != true {
			t.Fatalf("sleep task finished = %v, want true", content.GetTask().GetFinished())
		}

		testsupport.WaitForCondition(t, 5*time.Second, func() bool {
			session := mustStoredSession(t, f.h, f.implant.SessionID)
			return session.GetTimer().GetExpression() == "*/7 * * * * * *" && session.GetTimer().GetJitter() == 0.15
		}, "stored timer update after sleep")
		testsupport.WaitForCondition(t, 5*time.Second, func() bool {
			session := mustRuntimeSession(t, f.h, f.implant.SessionID)
			return session.GetTimer().GetExpression() == "*/7 * * * * * *" && session.GetTimer().GetJitter() == 0.15
		}, "runtime timer update after sleep")
		testsupport.WaitForCondition(t, 5*time.Second, func() bool {
			session := mustClientSession(t, f.client.Console, f.implant.SessionID)
			return session.GetTimer().GetExpression() == "*/7 * * * * * *" && session.GetTimer().GetJitter() == 0.15
		}, "client timer cache update after sleep")
	})

	t.Run("keepalive-enable-disable", func(t *testing.T) {
		enableTask := f.executeWait(t, consts.ModuleKeepalive, consts.ModuleKeepalive, "enable")
		enableContent := waitCommandTaskFinish(t, f.client, enableTask)
		if common := enableContent.GetSpite().GetCommon(); common == nil || len(common.GetBoolArray()) == 0 || !common.GetBoolArray()[0] {
			t.Fatalf("keepalive enable response = %#v, want bool_array[0]=true", enableContent.GetSpite().GetCommon())
		}
		testsupport.WaitForCondition(t, 5*time.Second, func() bool {
			enabled, err := f.h.RuntimeKeepaliveEnabled(f.implant.SessionID)
			return err == nil && enabled
		}, "runtime keepalive enabled")

		disableTask := f.executeWait(t, consts.ModuleKeepalive, consts.ModuleKeepalive, "disable")
		disableContent := waitCommandTaskFinish(t, f.client, disableTask)
		if common := disableContent.GetSpite().GetCommon(); common == nil || len(common.GetBoolArray()) == 0 || common.GetBoolArray()[0] {
			t.Fatalf("keepalive disable response = %#v, want bool_array[0]=false", disableContent.GetSpite().GetCommon())
		}
		testsupport.WaitForCondition(t, 5*time.Second, func() bool {
			enabled, err := f.h.RuntimeKeepaliveEnabled(f.implant.SessionID)
			return err == nil && !enabled
		}, "runtime keepalive disabled")
	})
}

func TestRealImplantCommandModuleManagementE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	requireSessionModules(
		t,
		f,
		consts.ModuleListModule,
		consts.ModuleRefreshModule,
	)

	listTask := f.executeWait(t, consts.ModuleListModule, consts.ModuleListModule)
	listContent := waitCommandTaskFinish(t, f.client, listTask)
	listModules := listContent.GetSpite().GetModules()
	if listModules == nil || len(listModules.GetModules()) == 0 {
		t.Fatalf("list_module response = %#v, want non-empty modules", listContent.GetSpite())
	}
	requireModulePresent(t, listModules.GetModules(), consts.ModulePwd)
	requireModulePresent(t, listModules.GetModules(), consts.ModuleSleep)
	requireModulePresent(t, listModules.GetModules(), consts.ModuleSwitch)

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		session := mustStoredSession(t, f.h, f.implant.SessionID)
		return len(session.GetModules()) >= len(listModules.GetModules())
	}, "stored module cache update after list_module")

	refreshTask := f.executeWait(t, consts.ModuleRefreshModule, consts.ModuleRefreshModule)
	refreshContent := waitCommandTaskFinish(t, f.client, refreshTask)
	refreshModules := refreshContent.GetSpite().GetModules()
	if refreshModules == nil || len(refreshModules.GetModules()) == 0 {
		t.Fatalf("refresh_module response = %#v, want non-empty modules", refreshContent.GetSpite())
	}
	requireModulePresent(t, refreshModules.GetModules(), consts.ModulePwd)
	requireModulePresent(t, refreshModules.GetModules(), consts.ModuleSleep)
	requireModulePresent(t, refreshModules.GetModules(), consts.ModuleSwitch)

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		session := mustClientSession(t, f.client.Console, f.implant.SessionID)
		return len(session.GetModules()) >= len(refreshModules.GetModules())
	}, "client module cache update after refresh_module")
}

func TestRealImplantCommandSwitchModuleE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	requireSessionModules(t, f, consts.ModuleSwitch, consts.ModulePwd)

	primary := proto.Clone(f.implant.Pipeline).(*clientpb.Pipeline)
	secondary := testsupport.NewRealTCPPipeline(
		t,
		f.implant.ListenerName,
		fmt.Sprintf("real-command-switch-%d", time.Now().UnixNano()),
	)
	registerAndStartPipeline(t, secondary)
	t.Cleanup(func() {
		stopPipeline(t, secondary)
	})

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, ok := f.client.Console.Pipelines[secondary.GetName()]
		return ok
	}, "client pipeline cache to include secondary pipeline")

	before := mustStoredSession(t, f.h, f.implant.SessionID)
	// Drive switch without command-layer --wait so the test can surface
	// missing task completion from the real implant instead of hanging here.
	switchTask := f.executeNoWait(
		t,
		consts.ModuleSwitch,
		consts.ModuleSwitch,
		"--pipeline",
		secondary.GetName(),
	)
	switchContent := waitCommandTaskFinish(t, f.client, switchTask)
	if switchContent.GetSpite().GetEmpty() == nil {
		t.Fatalf("switch response = %#v, want empty response", switchContent.GetSpite())
	}

	stopPipeline(t, primary)
	f.implant.Pipeline = nil

	deadline := time.Now().Add(15 * time.Second)
	var latest *clientpb.Session
	for time.Now().Before(deadline) {
		session, err := f.h.GetSession(f.implant.SessionID)
		if err == nil {
			latest = session
			if session.GetPipelineId() == secondary.GetName() && session.GetLastCheckin() > before.GetLastCheckin() {
				break
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	if latest == nil || latest.GetPipelineId() != secondary.GetName() || latest.GetLastCheckin() <= before.GetLastCheckin() {
		t.Fatalf(
			"switch did not migrate session to secondary pipeline: got pipeline=%q last_checkin=%d want pipeline=%q last_checkin>%d",
			func() string {
				if latest == nil {
					return ""
				}
				return latest.GetPipelineId()
			}(),
			func() int64 {
				if latest == nil {
					return 0
				}
				return latest.GetLastCheckin()
			}(),
			secondary.GetName(),
			before.GetLastCheckin(),
		)
	}

	testsupport.WaitForCondition(t, 10*time.Second, func() bool {
		session := mustClientSession(t, f.client.Console, f.implant.SessionID)
		return session.GetPipelineId() == secondary.GetName()
	}, "client session cache pipeline update after switch")

	pwdTask := f.executeWait(t, consts.ModulePwd, consts.ModulePwd)
	pwdContent := waitCommandTaskFinish(t, f.client, pwdTask)
	output := strings.TrimSpace(pwdContent.GetSpite().GetResponse().GetOutput())
	if output == "" {
		t.Fatal("pwd output should not be empty after switch")
	}
}

func TestRealImplantCommandFilesystemModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	requireSessionModules(
		t,
		f,
		consts.ModuleMkdir,
		consts.ModuleCd,
		consts.ModulePwd,
		consts.ModuleTouch,
		consts.ModuleCp,
		consts.ModuleCat,
		consts.ModuleMv,
		consts.ModuleLs,
		consts.ModuleRm,
	)

	tempResp := executeProgram(t, f, consts.ModuleAliasRun, "cmd.exe", "/c", "echo", "%TEMP%")
	tempDir := strings.TrimSpace(string(tempResp.GetStdout()))
	if tempDir == "" {
		t.Fatal("resolved TEMP directory is empty")
	}
	seedWorkdir := mustClientSession(t, f.client.Console, f.implant.SessionID).GetWorkdir()

	scratchDir := filepath.Join(tempDir, fmt.Sprintf("malice-real-fs-%d", time.Now().UnixNano()))
	emptyPath := filepath.Join(scratchDir, "empty.txt")
	copiedPath := filepath.Join(scratchDir, "copied.txt")
	movedPath := filepath.Join(scratchDir, "moved.txt")
	emptyName := filepath.Base(emptyPath)
	copiedName := filepath.Base(copiedPath)
	movedName := filepath.Base(movedPath)

	t.Cleanup(func() {
		_ = func() error {
			_ = f.executeWait(t, consts.ModuleCd, consts.ModuleCd, tempDir)
			_ = executeProgram(t, f, consts.ModuleAliasShell, fmt.Sprintf(`if exist "%s" rmdir /s /q "%s"`, scratchDir, scratchDir))
			return nil
		}()
	})

	t.Run("mkdir", func(t *testing.T) {
		task := f.executeWait(t, consts.ModuleMkdir, consts.ModuleMkdir, scratchDir)
		if content := waitCommandTaskFinish(t, f.client, task); !content.GetTask().GetFinished() {
			t.Fatalf("mkdir finished = %v, want true", content.GetTask().GetFinished())
		}
	})

	t.Run("cd-and-pwd", func(t *testing.T) {
		cdTask := f.executeWait(t, consts.ModuleCd, consts.ModuleCd, scratchDir)
		if content := waitCommandTaskFinish(t, f.client, cdTask); !content.GetTask().GetFinished() {
			t.Fatalf("cd finished = %v, want true", content.GetTask().GetFinished())
		}

		pwdTask := f.executeWait(t, consts.ModulePwd, consts.ModulePwd)
		pwdContent := waitCommandTaskFinish(t, f.client, pwdTask)
		got := strings.TrimSpace(pwdContent.GetSpite().GetResponse().GetOutput())
		if normalizeWindowsPath(got) != normalizeWindowsPath(scratchDir) {
			t.Fatalf("pwd output = %q, want %q", got, scratchDir)
		}
	})

	t.Run("touch", func(t *testing.T) {
		task := f.executeWait(t, consts.ModuleTouch, consts.ModuleTouch, emptyPath)
		if content := waitCommandTaskFinish(t, f.client, task); !content.GetTask().GetFinished() {
			t.Fatalf("touch finished = %v, want true", content.GetTask().GetFinished())
		}
	})

	t.Run("cp-cat-mv-rm-ls", func(t *testing.T) {
		seedListTask := f.executeWait(t, consts.ModuleLs, consts.ModuleLs, seedWorkdir)
		seedListContent := waitCommandTaskFinish(t, f.client, seedListTask)
		seedListResp := seedListContent.GetSpite().GetLsResponse()
		if seedListResp == nil {
			t.Fatal("seed ls response is nil")
		}

		var seedPath string
		for _, file := range seedListResp.GetFiles() {
			if strings.HasSuffix(strings.ToLower(file.GetName()), ".yaml") {
				seedPath = filepath.Join(seedWorkdir, file.GetName())
				break
			}
		}
		if seedPath == "" {
			t.Fatalf("unable to find text fixture file in %q from %#v", seedWorkdir, seedListResp.GetFiles())
		}

		seedCatTask := f.executeWait(t, consts.ModuleCat, consts.ModuleCat, seedPath)
		seedCatContent := waitCommandTaskFinish(t, f.client, seedCatTask)
		seedCatResp := seedCatContent.GetSpite().GetBinaryResponse()
		if seedCatResp == nil || len(seedCatResp.GetData()) == 0 {
			t.Fatalf("seed cat response = %#v, want non-empty data", seedCatResp)
		}

		cpTask := f.executeWait(t, consts.ModuleCp, consts.ModuleCp, seedPath, copiedPath)
		if content := waitCommandTaskFinish(t, f.client, cpTask); !content.GetTask().GetFinished() {
			t.Fatalf("cp finished = %v, want true", content.GetTask().GetFinished())
		}

		catTask := f.executeWait(t, consts.ModuleCat, consts.ModuleCat, copiedPath)
		catContent := waitCommandTaskFinish(t, f.client, catTask)
		catResp := catContent.GetSpite().GetBinaryResponse()
		if catResp == nil {
			t.Fatal("cat binary response is nil")
		}
		if string(catResp.GetData()) != string(seedCatResp.GetData()) {
			t.Fatalf("copied file data mismatch: got %d bytes, want original %d bytes", len(catResp.GetData()), len(seedCatResp.GetData()))
		}

		mvTask := f.executeWait(t, consts.ModuleMv, consts.ModuleMv, copiedPath, movedPath)
		if content := waitCommandTaskFinish(t, f.client, mvTask); !content.GetTask().GetFinished() {
			t.Fatalf("mv finished = %v, want true", content.GetTask().GetFinished())
		}

		lsTask := f.executeWait(t, consts.ModuleLs, consts.ModuleLs, scratchDir)
		lsContent := waitCommandTaskFinish(t, f.client, lsTask)
		lsResp := lsContent.GetSpite().GetLsResponse()
		if lsResp == nil || !lsResp.GetExists() {
			t.Fatalf("ls response = %#v, want existing directory", lsResp)
		}
		if normalizeWindowsPath(lsResp.GetPath()) != normalizeWindowsPath(scratchDir) {
			t.Fatalf("ls path = %q, want %q", lsResp.GetPath(), scratchDir)
		}
		requireLsContains(t, lsResp.GetFiles(), emptyName)
		requireLsContains(t, lsResp.GetFiles(), movedName)
		requireLsNotContains(t, lsResp.GetFiles(), copiedName)

		rmTask := f.executeWait(t, consts.ModuleRm, consts.ModuleRm, movedPath)
		if content := waitCommandTaskFinish(t, f.client, rmTask); !content.GetTask().GetFinished() {
			t.Fatalf("rm finished = %v, want true", content.GetTask().GetFinished())
		}

		lsAfterTask := f.executeWait(t, consts.ModuleLs, consts.ModuleLs, scratchDir)
		lsAfterContent := waitCommandTaskFinish(t, f.client, lsAfterTask)
		lsAfterResp := lsAfterContent.GetSpite().GetLsResponse()
		if lsAfterResp == nil {
			t.Fatal("ls after rm response is nil")
		}
		requireLsContains(t, lsAfterResp.GetFiles(), emptyName)
		requireLsNotContains(t, lsAfterResp.GetFiles(), movedName)
	})
}

func TestRealImplantCommandFileTransferModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	requireSessionModules(
		t,
		f,
		consts.ModuleUpload,
		consts.ModuleDownload,
		consts.ModuleMkdir,
		consts.ModuleCat,
	)

	tempResp := executeProgram(t, f, consts.ModuleAliasRun, "cmd.exe", "/c", "echo", "%TEMP%")
	tempDir := strings.TrimSpace(string(tempResp.GetStdout()))
	if tempDir == "" {
		t.Fatal("resolved TEMP directory is empty")
	}

	scratchDir := filepath.Join(tempDir, fmt.Sprintf("malice-real-file-%d", time.Now().UnixNano()))
	remotePath := filepath.Join(scratchDir, "uploaded.txt")
	localPath := filepath.Join(t.TempDir(), "upload.txt")
	uploadBody := []byte(fmt.Sprintf("real-file-transfer-%d", time.Now().UnixNano()))
	if err := os.WriteFile(localPath, uploadBody, 0o600); err != nil {
		t.Fatalf("write local upload fixture failed: %v", err)
	}

	t.Cleanup(func() {
		_ = executeProgram(t, f, consts.ModuleAliasShell, fmt.Sprintf(`if exist "%s" rmdir /s /q "%s"`, scratchDir, scratchDir))
	})

	mkdirTask := f.executeWait(t, consts.ModuleMkdir, consts.ModuleMkdir, scratchDir)
	if content := waitCommandTaskFinish(t, f.client, mkdirTask); !content.GetTask().GetFinished() {
		t.Fatalf("file transfer mkdir finished = %v, want true", content.GetTask().GetFinished())
	}

	t.Run("upload-and-cat", func(t *testing.T) {
		uploadTask := f.executeWait(
			t,
			consts.ModuleUpload,
			consts.ModuleUpload,
			localPath,
			remotePath,
			"--priv",
			"0600",
			"--hidden",
		)
		if content := waitCommandTaskFinish(t, f.client, uploadTask); !content.GetTask().GetFinished() {
			t.Fatalf("upload finished = %v, want true", content.GetTask().GetFinished())
		}

		catTask := f.executeWait(t, consts.ModuleCat, consts.ModuleCat, remotePath)
		catContent := waitCommandTaskFinish(t, f.client, catTask)
		catResp := catContent.GetSpite().GetBinaryResponse()
		if catResp == nil {
			t.Fatal("cat uploaded file response is nil")
		}
		if string(catResp.GetData()) != string(uploadBody) {
			t.Fatalf("uploaded remote content = %q, want %q", string(catResp.GetData()), string(uploadBody))
		}

		uploadContexts := getTaskContextsByType(t, f, consts.ContextUpload, uploadTask)
		if len(uploadContexts) != 1 {
			t.Fatalf("upload contexts = %d, want 1", len(uploadContexts))
		}
		uploadCtx, err := output.ToContext[*output.UploadContext](uploadContexts[0])
		if err != nil {
			t.Fatalf("parse upload context failed: %v", err)
		}
		if uploadCtx.TargetPath != remotePath {
			t.Fatalf("upload context target = %q, want %q", uploadCtx.TargetPath, remotePath)
		}
	})

	t.Run("download-roundtrip", func(t *testing.T) {
		downloadTask := f.executeWait(t, consts.ModuleDownload, consts.ModuleDownload, remotePath)
		if content := waitCommandTaskFinish(t, f.client, downloadTask); !content.GetTask().GetFinished() {
			t.Fatalf("download finished = %v, want true", content.GetTask().GetFinished())
		}

		downloadContexts := getTaskContextsByType(t, f, consts.ContextDownload, downloadTask)
		if len(downloadContexts) != 1 {
			t.Fatalf("download contexts = %d, want 1", len(downloadContexts))
		}
		downloadCtx, err := output.ToContext[*output.DownloadContext](downloadContexts[0])
		if err != nil {
			t.Fatalf("parse download context failed: %v", err)
		}
		if downloadCtx.TargetPath != remotePath {
			t.Fatalf("download context target = %q, want %q", downloadCtx.TargetPath, remotePath)
		}
		data, err := os.ReadFile(downloadCtx.FilePath)
		if err != nil {
			t.Fatalf("read downloaded file failed: %v", err)
		}
		if string(data) != string(uploadBody) {
			t.Fatalf("downloaded file content = %q, want %q", string(data), string(uploadBody))
		}
	})
}

func TestRealImplantCommandSystemInventoryModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	requireSessionModules(
		t,
		f,
		consts.ModuleEnv,
		consts.ModuleSetEnv,
		consts.ModuleUnsetEnv,
		consts.ModuleWhoami,
		consts.ModulePs,
		consts.ModuleNetstat,
	)

	t.Run("env-set-unset", func(t *testing.T) {
		envTask := f.executeWait(t, consts.ModuleEnv, consts.ModuleEnv)
		envContent := waitCommandTaskFinish(t, f.client, envTask)
		envResp := envContent.GetSpite().GetResponse()
		if envResp == nil || len(envResp.GetKv()) == 0 {
			t.Fatalf("env response = %#v, want non-empty kv", envResp)
		}
		if tempValue, ok := findEnvValue(envResp.GetKv(), "TEMP"); !ok || strings.TrimSpace(tempValue) == "" {
			t.Fatalf("env kv = %#v, want TEMP entry", envResp.GetKv())
		}

		key := fmt.Sprintf("MALICE_E2E_%d", time.Now().UnixNano())
		value := "real-env-roundtrip"

		setTask := f.executeWait(t, consts.ModuleSetEnv, consts.ModuleEnv, "set", key, value)
		if content := waitCommandTaskFinish(t, f.client, setTask); !content.GetTask().GetFinished() {
			t.Fatalf("setenv finished = %v, want true", content.GetTask().GetFinished())
		}

		envAfterSetTask := f.executeWait(t, consts.ModuleEnv, consts.ModuleEnv)
		envAfterSetContent := waitCommandTaskFinish(t, f.client, envAfterSetTask)
		got, ok := findEnvValue(envAfterSetContent.GetSpite().GetResponse().GetKv(), key)
		if !ok || got != value {
			t.Fatalf("env after set = %#v, want %s=%s", envAfterSetContent.GetSpite().GetResponse().GetKv(), key, value)
		}

		unsetTask := f.executeWait(t, consts.ModuleUnsetEnv, consts.ModuleEnv, "unset", key)
		if content := waitCommandTaskFinish(t, f.client, unsetTask); !content.GetTask().GetFinished() {
			t.Fatalf("unsetenv finished = %v, want true", content.GetTask().GetFinished())
		}

		envAfterUnsetTask := f.executeWait(t, consts.ModuleEnv, consts.ModuleEnv)
		envAfterUnsetContent := waitCommandTaskFinish(t, f.client, envAfterUnsetTask)
		if _, ok := findEnvValue(envAfterUnsetContent.GetSpite().GetResponse().GetKv(), key); ok {
			t.Fatalf("env after unset = %#v, want %q removed", envAfterUnsetContent.GetSpite().GetResponse().GetKv(), key)
		}
	})

	t.Run("whoami", func(t *testing.T) {
		task := f.executeWait(t, consts.ModuleWhoami, consts.ModuleWhoami)
		content := waitCommandTaskFinish(t, f.client, task)
		output := strings.TrimSpace(content.GetSpite().GetResponse().GetOutput())
		if output == "" {
			t.Fatal("whoami output should not be empty")
		}
	})

	t.Run("ps", func(t *testing.T) {
		task := f.executeWait(t, consts.ModulePs, consts.ModulePs)
		content := waitCommandTaskFinish(t, f.client, task)
		processes := content.GetSpite().GetPsResponse().GetProcesses()
		if len(processes) == 0 {
			t.Fatal("ps returned no processes")
		}

		session := mustClientSession(t, f.client.Console, f.implant.SessionID)
		foundSessionProcess := false
		for _, process := range processes {
			if process.GetPid() == session.GetProcess().GetPid() {
				foundSessionProcess = true
				break
			}
		}
		if !foundSessionProcess {
			t.Fatalf("ps result does not contain implant pid %d", session.GetProcess().GetPid())
		}
	})

	t.Run("netstat", func(t *testing.T) {
		task := f.executeWait(t, consts.ModuleNetstat, consts.ModuleNetstat)
		content := waitCommandTaskFinish(t, f.client, task)
		socks := content.GetSpite().GetNetstatResponse().GetSocks()
		if len(socks) == 0 {
			t.Fatal("netstat returned no sockets")
		}

		foundAddress := false
		for _, sock := range socks {
			if strings.TrimSpace(sock.GetLocalAddr()) != "" || strings.TrimSpace(sock.GetRemoteAddr()) != "" {
				foundAddress = true
				break
			}
		}
		if !foundAddress {
			t.Fatalf("netstat sockets = %#v, want at least one populated address", socks)
		}
	})

	t.Run("enum-drivers", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModuleEnumDrivers)

		task := f.executeWait(t, consts.ModuleEnumDrivers, consts.ModuleEnumDrivers)
		content := waitCommandTaskFinish(t, f.client, task)
		drives := content.GetSpite().GetEnumDriversResponse().GetDrives()
		if len(drives) == 0 {
			t.Fatal("enum_drivers returned no drives")
		}
		if strings.TrimSpace(drives[0].GetPath()) == "" {
			t.Fatalf("enum_drivers first drive = %#v, want non-empty path", drives[0])
		}
	})
}

func TestRealImplantCommandSystemActionModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	requireSessionModules(
		t,
		f,
		consts.ModuleKill,
		consts.ModuleBypass,
		consts.ModulePs,
	)

	t.Run("kill", func(t *testing.T) {
		spawnResp := executeProgram(
			t,
			f,
			consts.ModuleAliasRun,
			"powershell.exe",
			"-NoProfile",
			"-NonInteractive",
			"-Command",
			"$p = Start-Process ping -ArgumentList '127.0.0.1 -n 30' -WindowStyle Hidden -PassThru; $p.Id",
		)
		pid := parseUint32FromOutput(string(spawnResp.GetStdout()))
		if pid == 0 {
			t.Fatalf("spawned pid parse failed from %q", string(spawnResp.GetStdout()))
		}

		t.Cleanup(func() {
			_ = executeProgram(t, f, consts.ModuleAliasRun, "cmd.exe", "/c", "taskkill", "/PID", fmt.Sprintf("%d", pid), "/F")
		})

		killTask := f.executeWait(t, consts.ModuleKill, consts.ModuleKill, fmt.Sprintf("%d", pid))
		if content := waitCommandTaskFinish(t, f.client, killTask); !content.GetTask().GetFinished() {
			t.Fatalf("kill finished = %v, want true", content.GetTask().GetFinished())
		}

		psTask := f.executeWait(t, consts.ModulePs, consts.ModulePs)
		psContent := waitCommandTaskFinish(t, f.client, psTask)
		for _, process := range psContent.GetSpite().GetPsResponse().GetProcesses() {
			if process.GetPid() == pid {
				t.Fatalf("killed pid %d still present in ps output", pid)
			}
		}
	})

	t.Run("bypass", func(t *testing.T) {
		task := f.executeWait(t, consts.ModuleBypass, consts.ModuleBypass, "--amsi", "--etw")
		if content := waitCommandTaskFinish(t, f.client, task); !content.GetTask().GetFinished() {
			t.Fatalf("bypass finished = %v, want true", content.GetTask().GetFinished())
		}
	})
}

func TestRealImplantCommandTaskControlE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	requireSessionModules(
		t,
		f,
		consts.ModuleExecute,
		consts.ModuleListTask,
		consts.ModuleQueryTask,
		consts.ModuleCancelTask,
	)

	targetTask := f.executeNoWait(
		t,
		consts.ModuleExecute,
		consts.ModuleAliasRun,
		"cmd.exe",
		"/c",
		"ping -n 20 127.0.0.1 > nul",
	)

	time.Sleep(2 * time.Second)

	listTask := f.executeWait(t, consts.ModuleListTask, consts.ModuleListTask)
	listContent := waitCommandTaskFinish(t, f.client, listTask)
	taskList := listContent.GetSpite().GetTaskList()
	if taskList == nil {
		t.Fatal("list_task response is nil")
	}
	listed := findTaskInfo(taskList.GetTasks(), targetTask.GetTaskId())
	if listed == nil {
		t.Fatalf("list_task tasks = %#v, want task %d", taskList.GetTasks(), targetTask.GetTaskId())
	}

	queryTask := f.executeWait(
		t,
		consts.ModuleQueryTask,
		consts.ModuleQueryTask,
		fmt.Sprintf("%d", targetTask.GetTaskId()),
	)
	queryContent := waitCommandTaskFinish(t, f.client, queryTask)
	taskInfo := queryContent.GetSpite().GetTaskInfo()
	if taskInfo == nil {
		t.Fatal("query_task response is nil")
	}
	if taskInfo.GetTaskId() != targetTask.GetTaskId() {
		t.Fatalf("query_task id = %d, want %d", taskInfo.GetTaskId(), targetTask.GetTaskId())
	}

	cancelTask := f.executeWait(
		t,
		consts.ModuleCancelTask,
		consts.ModuleCancelTask,
		fmt.Sprintf("%d", targetTask.GetTaskId()),
	)
	cancelContent := waitCommandTaskFinish(t, f.client, cancelTask)
	if cancelContent.GetSpite().GetEmpty() == nil {
		t.Fatalf("cancel_task response = %#v, want empty response", cancelContent.GetSpite())
	}

	time.Sleep(2 * time.Second)

	listAfterCancel := f.executeWait(t, consts.ModuleListTask, consts.ModuleListTask)
	listAfterCancelContent := waitCommandTaskFinish(t, f.client, listAfterCancel)
	if taskList := listAfterCancelContent.GetSpite().GetTaskList(); taskList != nil {
		if found := findTaskInfo(taskList.GetTasks(), targetTask.GetTaskId()); found != nil {
			t.Fatalf("canceled task %d still listed after cancel: %#v", targetTask.GetTaskId(), taskList.GetTasks())
		}
	}

	runtimeTask, err := f.h.GetRuntimeTask(f.implant.SessionID, targetTask.GetTaskId())
	if err == nil && !runtimeTask.GetFinished() {
		t.Fatalf("canceled target task still unfinished in server runtime: %#v", runtimeTask)
	}
}

func TestRealImplantCommandAddonModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	requireSessionModules(
		t,
		f,
		consts.ModuleWhoami,
		consts.ModuleExecuteExe,
	)

	systemRoot := os.Getenv("WINDIR")
	if strings.TrimSpace(systemRoot) == "" {
		systemRoot = `C:\Windows`
	}
	addonPath := filepath.Join(systemRoot, "System32", "whoami.exe")
	if _, err := os.Stat(addonPath); err != nil {
		t.Skipf("addon sample %s not available: %v", addonPath, err)
	}

	addonName := fmt.Sprintf("whoami-addon-%d", time.Now().UnixNano())

	loadTask := f.executeWait(
		t,
		consts.ModuleLoadAddon,
		consts.ModuleLoadAddon,
		"--name",
		addonName,
		"--module",
		consts.ModuleExecuteExe,
		addonPath,
	)
	loadContent := waitCommandTaskFinish(t, f.client, loadTask)
	if loadContent.GetSpite().GetEmpty() == nil {
		t.Fatalf("load_addon response = %#v, want empty response", loadContent.GetSpite())
	}

	clientAddon := waitForClientAddon(t, f, addonName)
	if clientAddon.GetDepend() != consts.ModuleExecuteExe {
		t.Fatalf("client addon depend = %q, want %q", clientAddon.GetDepend(), consts.ModuleExecuteExe)
	}
	storedAddon := waitForStoredAddon(t, f, addonName)
	if storedAddon.GetDepend() != consts.ModuleExecuteExe {
		t.Fatalf("stored addon depend = %q, want %q", storedAddon.GetDepend(), consts.ModuleExecuteExe)
	}

	testsupport.WaitForCondition(t, 10*time.Second, func() bool {
		return implantMenuHasCommand(f.client.Console.ImplantMenu(), addonName)
	}, "dynamic addon command "+addonName)

	listTask := f.executeWait(t, consts.ModuleListAddon, consts.ModuleListAddon)
	listContent := waitCommandTaskFinish(t, f.client, listTask)
	addons := listContent.GetSpite().GetAddons()
	if addons == nil {
		t.Fatal("list_addon response is nil")
	}
	listedAddon := findAddon(addons.GetAddons(), addonName)
	if listedAddon == nil {
		t.Fatalf("list_addon result = %#v, want addon %q", addons.GetAddons(), addonName)
	}

	whoamiTask := f.executeWait(t, consts.ModuleWhoami, consts.ModuleWhoami)
	whoamiContent := waitCommandTaskFinish(t, f.client, whoamiTask)
	whoamiOutput := strings.ToLower(strings.TrimSpace(whoamiContent.GetSpite().GetResponse().GetOutput()))
	if whoamiOutput == "" {
		t.Fatal("whoami output should not be empty")
	}

	executeTask := f.executeWait(
		t,
		consts.ModuleExecuteAddon,
		consts.ModuleExecuteAddon,
		addonName,
	)
	executeContent := waitCommandTaskFinish(t, f.client, executeTask)
	executeOutput := strings.ToLower(strings.TrimSpace(string(executeContent.GetSpite().GetBinaryResponse().GetData())))
	if executeOutput == "" {
		t.Fatal("execute_addon output should not be empty")
	}
	if !strings.Contains(executeOutput, whoamiOutput) {
		t.Fatalf("execute_addon output = %q, want contains %q", executeOutput, whoamiOutput)
	}

	dynamicTask := f.executeWait(t, consts.ModuleExecuteAddon, addonName)
	dynamicContent := waitCommandTaskFinish(t, f.client, dynamicTask)
	dynamicOutput := strings.ToLower(strings.TrimSpace(string(dynamicContent.GetSpite().GetBinaryResponse().GetData())))
	if dynamicOutput == "" {
		t.Fatal("dynamic addon command output should not be empty")
	}
	if !strings.Contains(dynamicOutput, whoamiOutput) {
		t.Fatalf("dynamic addon output = %q, want contains %q", dynamicOutput, whoamiOutput)
	}
}

func TestRealImplantCommandTokenModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	t.Run("runas-invalid-credentials", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModuleRunas)

		task, err := f.executeMaybeWait(
			t,
			consts.ModuleRunas,
			"--username",
			"malice_nonexistent_user",
			"--domain",
			".",
			"--password",
			"malice_bad_password",
			"--path",
			"cmd.exe",
			"--args",
			"/c whoami",
		)
		if task == nil || task.GetType() != consts.ModuleRunas {
			t.Fatalf("runas task = %#v, want non-nil task type %q", task, consts.ModuleRunas)
		}
		if err == nil {
			content := waitCommandTaskFinish(t, f.client, task)
			resp := content.GetSpite().GetExecResponse()
			if resp == nil {
				t.Fatal("runas exec response is nil")
			}
			if strings.TrimSpace(string(resp.GetStdout())) == "" && strings.TrimSpace(string(resp.GetStderr())) == "" {
				t.Fatalf("runas response = %#v, want stdout/stderr diagnostics", resp)
			}
			return
		}
		if strings.TrimSpace(err.Error()) == "" {
			t.Fatal("runas error should include diagnostics")
		}
	})

	t.Run("privs", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModulePrivs)

		task := f.executeWait(t, consts.ModulePrivs, consts.ModulePrivs)
		content := waitCommandTaskFinish(t, f.client, task)
		resp := content.GetSpite().GetResponse()
		if resp == nil {
			t.Fatal("privs response is nil")
		}
		if len(resp.GetArray()) == 0 && len(resp.GetKv()) == 0 && strings.TrimSpace(resp.GetOutput()) == "" {
			t.Fatalf("privs response = %#v, want non-empty privileges data", resp)
		}
	})

	t.Run("getsystem-attempt", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModuleGetSystem)

		task, err := f.executeMaybeWait(t, consts.ModuleGetSystem, consts.ModuleGetSystem)
		if task == nil || task.GetType() != consts.ModuleGetSystem {
			t.Fatalf("getsystem task = %#v, want non-nil task type %q", task, consts.ModuleGetSystem)
		}
		if err == nil {
			content := waitCommandTaskFinish(t, f.client, task)
			resp := content.GetSpite().GetResponse()
			if resp == nil || strings.TrimSpace(resp.GetOutput()) == "" {
				t.Fatalf("getsystem success response = %#v, want diagnostic output", resp)
			}
			return
		}
		if strings.TrimSpace(err.Error()) == "" {
			t.Fatal("getsystem error should include diagnostics")
		}
	})

	t.Run("rev2self", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModuleRev2Self, consts.ModuleWhoami)

		beforeTask := f.executeWait(t, consts.ModuleWhoami, consts.ModuleWhoami)
		beforeContent := waitCommandTaskFinish(t, f.client, beforeTask)
		before := strings.TrimSpace(beforeContent.GetSpite().GetResponse().GetOutput())

		revTask := f.executeWait(t, consts.ModuleRev2Self, consts.ModuleRev2Self)
		if content := waitCommandTaskFinish(t, f.client, revTask); !content.GetTask().GetFinished() {
			t.Fatalf("rev2self finished = %v, want true", content.GetTask().GetFinished())
		}

		afterTask := f.executeWait(t, consts.ModuleWhoami, consts.ModuleWhoami)
		afterContent := waitCommandTaskFinish(t, f.client, afterTask)
		after := strings.TrimSpace(afterContent.GetSpite().GetResponse().GetOutput())
		if before == "" || after == "" {
			t.Fatalf("whoami before/after = %q/%q, want non-empty", before, after)
		}
	})
}

func TestRealImplantCommandWindowsManagementModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	t.Run("registry-roundtrip", func(t *testing.T) {
		requireSessionModules(
			t,
			f,
			consts.ModuleRegAdd,
			consts.ModuleRegQuery,
			consts.ModuleRegDelete,
			consts.ModuleRegListKey,
			consts.ModuleRegListValue,
		)

		rootKey := `HKCU\Software\MaliceNetworkE2E`
		keyName := fmt.Sprintf("Reg-%d", time.Now().UnixNano())
		fullKey := rootKey + `\` + keyName
		stringValue := "Greeting"
		stringData := fmt.Sprintf("hello-reg-e2e-%d", time.Now().UnixNano())
		dwordValue := "Counter"

		addStringTask := f.executeWait(
			t,
			consts.ModuleRegAdd,
			consts.CommandReg,
			"add",
			fullKey,
			"--value",
			stringValue,
			"--type",
			"REG_SZ",
			"--data",
			stringData,
		)
		if content := waitCommandTaskFinish(t, f.client, addStringTask); !content.GetTask().GetFinished() {
			t.Fatalf("reg add string finished = %v, want true", content.GetTask().GetFinished())
		}

		addDwordTask := f.executeWait(
			t,
			consts.ModuleRegAdd,
			consts.CommandReg,
			"add",
			fullKey,
			"--value",
			dwordValue,
			"--type",
			"REG_DWORD",
			"--data",
			"7",
		)
		if content := waitCommandTaskFinish(t, f.client, addDwordTask); !content.GetTask().GetFinished() {
			t.Fatalf("reg add dword finished = %v, want true", content.GetTask().GetFinished())
		}

		queryStringTask := f.executeWait(t, consts.ModuleRegQuery, consts.CommandReg, "query", fullKey, stringValue)
		queryStringContent := waitCommandTaskFinish(t, f.client, queryStringTask)
		queryStringOutput := strings.TrimSpace(queryStringContent.GetSpite().GetResponse().GetOutput())
		if !strings.Contains(queryStringOutput, stringData) {
			t.Fatalf("reg query string output = %q, want contains %q", queryStringOutput, stringData)
		}

		queryDwordTask := f.executeWait(t, consts.ModuleRegQuery, consts.CommandReg, "query", fullKey, dwordValue)
		queryDwordContent := waitCommandTaskFinish(t, f.client, queryDwordTask)
		queryDwordOutput := strings.TrimSpace(queryDwordContent.GetSpite().GetResponse().GetOutput())
		if !strings.Contains(queryDwordOutput, "7") {
			t.Fatalf("reg query dword output = %q, want contains 7", queryDwordOutput)
		}

		listValueTask := f.executeWait(t, consts.ModuleRegListValue, consts.CommandReg, "list_value", fullKey)
		listValueContent := waitCommandTaskFinish(t, f.client, listValueTask)
		values := listValueContent.GetSpite().GetResponse().GetKv()
		if got := values[stringValue]; !strings.Contains(got, stringData) {
			t.Fatalf("reg list_value %q = %q, want contains %q", stringValue, got, stringData)
		}
		if got := values[dwordValue]; !strings.Contains(got, "7") {
			t.Fatalf("reg list_value %q = %q, want contains 7", dwordValue, got)
		}

		listKeyTask := f.executeWait(t, consts.ModuleRegListKey, consts.CommandReg, "list_key", rootKey)
		listKeyContent := waitCommandTaskFinish(t, f.client, listKeyTask)
		foundKey := false
		for _, subkey := range listKeyContent.GetSpite().GetResponse().GetArray() {
			if subkey == keyName {
				foundKey = true
				break
			}
		}
		if !foundKey {
			t.Fatalf("reg list_key result = %v, want contains %q", listKeyContent.GetSpite().GetResponse().GetArray(), keyName)
		}

		deleteStringTask := f.executeWait(t, consts.ModuleRegDelete, consts.CommandReg, "delete", fullKey, stringValue)
		if content := waitCommandTaskFinish(t, f.client, deleteStringTask); !content.GetTask().GetFinished() {
			t.Fatalf("reg delete string finished = %v, want true", content.GetTask().GetFinished())
		}

		deleteDwordTask := f.executeWait(t, consts.ModuleRegDelete, consts.CommandReg, "delete", fullKey, dwordValue)
		if content := waitCommandTaskFinish(t, f.client, deleteDwordTask); !content.GetTask().GetFinished() {
			t.Fatalf("reg delete dword finished = %v, want true", content.GetTask().GetFinished())
		}

		listValueAfterDeleteTask := f.executeWait(t, consts.ModuleRegListValue, consts.CommandReg, "list_value", fullKey)
		listValueAfterDeleteContent := waitCommandTaskFinish(t, f.client, listValueAfterDeleteTask)
		afterDelete := listValueAfterDeleteContent.GetSpite().GetResponse().GetKv()
		if _, ok := afterDelete[stringValue]; ok {
			t.Fatalf("reg list_value after delete still contains %q: %v", stringValue, afterDelete)
		}
		if _, ok := afterDelete[dwordValue]; ok {
			t.Fatalf("reg list_value after delete still contains %q: %v", dwordValue, afterDelete)
		}
	})

	t.Run("service-list-query", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModuleServiceList, consts.ModuleServiceQuery)

		listTask := f.executeWait(t, consts.ModuleServiceList, consts.CommandService, "list")
		listContent := waitCommandTaskFinish(t, f.client, listTask)
		services := listContent.GetSpite().GetServicesResponse().GetServices()
		if len(services) == 0 {
			t.Fatal("service list returned no services")
		}

		selected := pickService(services)
		if selected.Name == "" {
			t.Fatalf("unable to select service from list response: %#v", services)
		}

		queryTask := f.executeWait(t, consts.ModuleServiceQuery, consts.CommandService, "query", selected.Name)
		queryContent := waitCommandTaskFinish(t, f.client, queryTask)
		serviceResp := queryContent.GetSpite().GetServiceResponse()
		if serviceResp == nil || serviceResp.GetConfig() == nil {
			t.Fatalf("service query response = %#v, want non-nil config", serviceResp)
		}
		if serviceResp.GetConfig().GetName() != selected.Name {
			t.Fatalf("service query name = %q, want %q", serviceResp.GetConfig().GetName(), selected.Name)
		}
		if selected.DisplayName != "" && serviceResp.GetConfig().GetDisplayName() != selected.DisplayName {
			t.Fatalf("service query display name = %q, want %q", serviceResp.GetConfig().GetDisplayName(), selected.DisplayName)
		}
		if serviceResp.GetConfig().GetExecutablePath() == "" {
			t.Fatalf("service query executable path = %q, want non-empty", serviceResp.GetConfig().GetExecutablePath())
		}
	})

	t.Run("taskschd-list-query", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModuleTaskSchdList, consts.ModuleTaskSchdQuery)

		listTask := f.executeWait(t, consts.ModuleTaskSchdList, consts.CommandTaskSchd, "list")
		listContent := waitCommandTaskFinish(t, f.client, listTask)
		schedules := listContent.GetSpite().GetSchedulesResponse().GetSchedules()
		if len(schedules) == 0 {
			t.Fatal("taskschd list returned no schedules")
		}

		selected := pickSchedule(schedules)
		if selected.Name == "" || selected.Path == "" {
			t.Fatalf("unable to select schedule from list response: %#v", schedules)
		}

		queryFolder := selected.Path
		if strings.HasSuffix(selected.Path, `\`+selected.Name) {
			t.Errorf("taskschd list path = %q includes task name %q; query expects folder path semantics", selected.Path, selected.Name)
			queryFolder = deriveTaskFolder(selected.Path, selected.Name)
		}

		queryTask := f.executeWait(
			t,
			consts.ModuleTaskSchdQuery,
			consts.CommandTaskSchd,
			"query",
			selected.Name,
			"--task_folder",
			queryFolder,
		)
		queryContent := waitCommandTaskFinish(t, f.client, queryTask)
		scheduleResp := queryContent.GetSpite().GetScheduleResponse()
		if scheduleResp == nil {
			t.Fatal("taskschd query response is nil")
		}
		if scheduleResp.GetName() != selected.Name {
			t.Fatalf("taskschd query name = %q, want %q", scheduleResp.GetName(), selected.Name)
		}
		if scheduleResp.GetPath() != selected.Path {
			t.Fatalf("taskschd query path = %q, want %q", scheduleResp.GetPath(), selected.Path)
		}
	})

	t.Run("wmi-query", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModuleWmiQuery)

		queryTask := f.executeWait(
			t,
			consts.ModuleWmiQuery,
			consts.ModuleWmiQuery,
			"--namespace",
			`root\cimv2`,
			"--args",
			"SELECT Caption FROM Win32_OperatingSystem",
		)
		queryContent := waitCommandTaskFinish(t, f.client, queryTask)
		kv := queryContent.GetSpite().GetResponse().GetKv()
		if !strings.Contains(kv["Caption"], "Windows") {
			t.Fatalf("wmi query caption = %q, want non-empty", kv["Caption"])
		}
	})

	t.Run("wmi-execute", func(t *testing.T) {
		requireSessionModules(t, f, consts.ModuleWmiExec)

		commandLine := "cmd.exe /c echo wmi-e2e"

		execTask := f.executeWait(
			t,
			consts.ModuleWmiExec,
			consts.ModuleWmiExec,
			"--namespace",
			`root\cimv2`,
			"--class_name",
			"Win32_Process",
			"--method_name",
			"Create",
			"--params",
			"CommandLine="+commandLine,
		)
		execContent := waitCommandTaskFinish(t, f.client, execTask)
		resp := execContent.GetSpite().GetResponse()
		if resp == nil {
			t.Fatal("wmi execute response is nil")
		}
		if !strings.Contains(resp.GetOutput(), "Executed Win32_Process::Create") {
			t.Fatalf("wmi execute output = %q, want Executed Win32_Process::Create", resp.GetOutput())
		}
		if returnValue := strings.TrimSpace(resp.GetKv()["ReturnValue"]); returnValue != "" && returnValue != "0" {
			t.Fatalf("wmi execute ReturnValue = %q, want 0", returnValue)
		}
		if strings.TrimSpace(resp.GetKv()["ReturnValue"]) == "" && strings.TrimSpace(resp.GetKv()["ProcessId"]) == "" {
			t.Fatalf("wmi execute kv = %v, want ReturnValue or ProcessId", resp.GetKv())
		}
	})
}

func TestRealImplantCommandWindowsPrivilegedModulesE2E(t *testing.T) {
	f := newRealCommandFixture(t)

	t.Run("taskschd-create-query-run-delete", func(t *testing.T) {
		requireSessionModules(
			t,
			f,
			consts.ModuleTaskSchdCreate,
			consts.ModuleTaskSchdQuery,
			consts.ModuleTaskSchdRun,
			consts.ModuleTaskSchdDelete,
		)
		if !isElevatedSession(t, f) {
			t.Skip("taskschd lifecycle requires elevated implant token")
		}

		taskName := fmt.Sprintf("MaliceE2E-%d", time.Now().UnixNano())
		taskFolder := `\`
		taskPath := `C:\Windows\System32\whoami.exe`
		startBoundary := "2030-01-01T00:00:00"

		createTask := f.executeWait(
			t,
			consts.ModuleTaskSchdCreate,
			consts.CommandTaskSchd,
			"create",
			"--name",
			taskName,
			"--path",
			taskPath,
			"--task_folder",
			taskFolder,
			"--trigger_type",
			"daily",
			"--start_boundary",
			startBoundary,
		)
		if content := waitCommandTaskFinish(t, f.client, createTask); !content.GetTask().GetFinished() {
			t.Fatalf("taskschd create finished = %v, want true", content.GetTask().GetFinished())
		}

		queryTask := f.executeWait(
			t,
			consts.ModuleTaskSchdQuery,
			consts.CommandTaskSchd,
			"query",
			taskName,
			"--task_folder",
			taskFolder,
		)
		queryContent := waitCommandTaskFinish(t, f.client, queryTask)
		scheduleResp := queryContent.GetSpite().GetScheduleResponse()
		if scheduleResp == nil {
			t.Fatal("taskschd query created response is nil")
		}
		if scheduleResp.GetName() != taskName || scheduleResp.GetPath() != taskFolder {
			t.Fatalf("taskschd query created schedule = %#v, want name=%q path=%q", scheduleResp, taskName, taskFolder)
		}
		if scheduleResp.GetExecutablePath() != taskPath {
			t.Fatalf("taskschd query executable path = %q, want %q", scheduleResp.GetExecutablePath(), taskPath)
		}

		runTask := f.executeWait(
			t,
			consts.ModuleTaskSchdRun,
			consts.CommandTaskSchd,
			"run",
			taskName,
			"--task_folder",
			taskFolder,
		)
		if content := waitCommandTaskFinish(t, f.client, runTask); !content.GetTask().GetFinished() {
			t.Fatalf("taskschd run finished = %v, want true", content.GetTask().GetFinished())
		}

		deleteTask := f.executeWait(
			t,
			consts.ModuleTaskSchdDelete,
			consts.CommandTaskSchd,
			"delete",
			taskName,
			"--task_folder",
			taskFolder,
		)
		if content := waitCommandTaskFinish(t, f.client, deleteTask); !content.GetTask().GetFinished() {
			t.Fatalf("taskschd delete finished = %v, want true", content.GetTask().GetFinished())
		}
	})
}
