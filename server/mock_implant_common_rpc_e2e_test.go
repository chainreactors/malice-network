//go:build mockimplant

package main

import (
	"context"
	"os"
	stdpath "path"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/testsupport"
	"google.golang.org/grpc/metadata"
)

type mockRPCFixture struct {
	h       *testsupport.ControlPlaneHarness
	mock    *testsupport.MockImplant
	lib     *testsupport.MockScenarioLibrary
	rpc     clientrpc.MaliceRPCClient
	session context.Context
}

func newMockRPCFixture(t *testing.T) *mockRPCFixture {
	t.Helper()

	h := testsupport.NewControlPlaneHarness(t)
	mock := testsupport.NewMockImplant(t, h, h.NewTCPPipeline(t, "mock-implant-common-pipe"))
	lib := testsupport.NewMockScenarioLibrary()
	lib.Install(mock)
	if err := mock.Start(); err != nil {
		t.Fatalf("mock implant start failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return &mockRPCFixture{
		h:    h,
		mock: mock,
		lib:  lib,
		rpc:  clientrpc.NewMaliceRPCClient(conn),
		session: metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
			"session_id", mock.SessionID,
			"callee", consts.CalleeCMD,
		)),
	}
}

func waitTaskFinish(t *testing.T, rpc clientrpc.MaliceRPCClient, sessionID string, taskID uint32) *clientpb.TaskContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	content, err := rpc.WaitTaskFinish(ctx, &clientpb.Task{
		SessionId: sessionID,
		TaskId:    taskID,
	})
	if err != nil {
		t.Fatalf("WaitTaskFinish(%d) failed: %v", taskID, err)
	}
	if content == nil || content.Task == nil || content.Spite == nil {
		t.Fatalf("WaitTaskFinish(%d) returned incomplete content: %#v", taskID, content)
	}
	return content
}

func waitModuleRequest(t *testing.T, mock *testsupport.MockImplant, module string, before int) *clientpb.SpiteRequest {
	t.Helper()

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return len(mock.RequestsByName(module)) == before+1
	}, "mock implant request "+module)

	request := mock.LastRequest(module)
	if request == nil {
		t.Fatalf("last request for %s is nil", module)
	}
	return request
}

func requireModule(t *testing.T, modules []string, want string) {
	t.Helper()
	for _, module := range modules {
		if module == want {
			return
		}
	}
	t.Fatalf("module list %v does not contain %q", modules, want)
}

func requireAddon(t *testing.T, addons []*implantpb.Addon, want string) {
	t.Helper()
	for _, addon := range addons {
		if addon.GetName() == want {
			return
		}
	}
	t.Fatalf("addon list does not contain %q", want)
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// Normalise to forward slashes for path.Clean, then back to backslashes.
	p = strings.ReplaceAll(p, `\`, "/")
	p = stdpath.Clean(p)
	p = strings.ReplaceAll(p, "/", `\`)
	if len(p) == 2 && strings.HasSuffix(p, ":") {
		p += `\`
	}
	return strings.ToLower(p)
}

func requireFileInfo(t *testing.T, files []*implantpb.FileInfo, want string) {
	t.Helper()
	want = strings.ToLower(want)
	for _, file := range files {
		if strings.ToLower(file.GetName()) == want {
			return
		}
	}
	t.Fatalf("ls response does not contain %q: %#v", want, files)
}

func requireNoFileInfo(t *testing.T, files []*implantpb.FileInfo, want string) {
	t.Helper()
	want = strings.ToLower(want)
	for _, file := range files {
		if strings.ToLower(file.GetName()) == want {
			t.Fatalf("ls response unexpectedly contains %q: %#v", want, files)
		}
	}
}

func serviceByName(services []*implantpb.Service, want string) *implantpb.Service {
	want = strings.ToLower(want)
	for _, service := range services {
		if strings.ToLower(service.GetConfig().GetName()) == want {
			return service
		}
	}
	return nil
}

func scheduleByName(schedules []*implantpb.TaskSchedule, want string) *implantpb.TaskSchedule {
	want = strings.ToLower(want)
	for _, schedule := range schedules {
		if strings.ToLower(schedule.GetName()) == want {
			return schedule
		}
	}
	return nil
}

func TestMockImplantCommonQueryRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	type testCase struct {
		name          string
		module        string
		invoke        func() (*clientpb.Task, error)
		assertRequest func(*testing.T, *clientpb.SpiteRequest)
		assertContent func(*testing.T, *clientpb.TaskContext)
		assertAfter   func(*testing.T)
	}

	cases := []testCase{
		{
			name:   "info",
			module: consts.ModuleSysInfo,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Info(f.session, &implantpb.Request{Name: consts.ModuleSysInfo})
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				if content.GetSpite().GetSysinfo().GetWorkdir() != f.lib.WorkDir {
					t.Fatalf("info workdir = %q, want %q", content.GetSpite().GetSysinfo().GetWorkdir(), f.lib.WorkDir)
				}
			},
			assertAfter: func(t *testing.T) {
				session, err := f.h.GetSession(f.mock.SessionID)
				if err != nil {
					t.Fatalf("GetSession failed: %v", err)
				}
				if session.GetWorkdir() != f.lib.WorkDir {
					t.Fatalf("saved session workdir = %q, want %q", session.GetWorkdir(), f.lib.WorkDir)
				}
			},
		},
		{
			name:   "ping",
			module: consts.ModulePing,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Ping(f.session, &implantpb.Ping{Nonce: 77})
			},
			assertRequest: func(t *testing.T, request *clientpb.SpiteRequest) {
				if request.GetSpite().GetPing().GetNonce() != 77 {
					t.Fatalf("ping nonce = %d, want 77", request.GetSpite().GetPing().GetNonce())
				}
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				if content.GetSpite().GetPing().GetNonce() != 77 {
					t.Fatalf("ping response nonce = %d, want 77", content.GetSpite().GetPing().GetNonce())
				}
			},
		},
		{
			name:   "pwd",
			module: consts.ModulePwd,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Pwd(f.session, &implantpb.Request{Name: consts.ModulePwd})
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				if content.GetSpite().GetResponse().GetOutput() != f.lib.WorkDir {
					t.Fatalf("pwd output = %q, want %q", content.GetSpite().GetResponse().GetOutput(), f.lib.WorkDir)
				}
			},
		},
		{
			name:   "ls",
			module: consts.ModuleLs,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Ls(f.session, &implantpb.Request{Name: consts.ModuleLs, Input: f.lib.WorkDir})
			},
			assertRequest: func(t *testing.T, request *clientpb.SpiteRequest) {
				if got := request.GetSpite().GetRequest().GetInput(); got != f.lib.WorkDir {
					t.Fatalf("ls input = %q, want %q", got, f.lib.WorkDir)
				}
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				resp := content.GetSpite().GetLsResponse()
				if !resp.GetExists() {
					t.Fatal("ls response should exist")
				}
				if len(resp.GetFiles()) < 2 {
					t.Fatalf("ls file count = %d, want >=2", len(resp.GetFiles()))
				}
			},
		},
		{
			name:   "cat",
			module: consts.ModuleCat,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Cat(f.session, &implantpb.Request{Name: consts.ModuleCat, Input: f.lib.NotesPath})
			},
			assertRequest: func(t *testing.T, request *clientpb.SpiteRequest) {
				if got := request.GetSpite().GetRequest().GetInput(); got != f.lib.NotesPath {
					t.Fatalf("cat input = %q, want %q", got, f.lib.NotesPath)
				}
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				if !strings.Contains(string(content.GetSpite().GetBinaryResponse().GetData()), "rpc coverage") {
					t.Fatalf("cat content = %q, want mock file payload", string(content.GetSpite().GetBinaryResponse().GetData()))
				}
			},
		},
		{
			name:   "ps",
			module: consts.ModulePs,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Ps(f.session, &implantpb.Request{Name: consts.ModulePs})
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				if len(content.GetSpite().GetPsResponse().GetProcesses()) < 3 {
					t.Fatalf("ps count = %d, want >=3", len(content.GetSpite().GetPsResponse().GetProcesses()))
				}
			},
		},
		{
			name:   "netstat",
			module: consts.ModuleNetstat,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Netstat(f.session, &implantpb.Request{Name: consts.ModuleNetstat})
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				if len(content.GetSpite().GetNetstatResponse().GetSocks()) < 2 {
					t.Fatalf("netstat count = %d, want >=2", len(content.GetSpite().GetNetstatResponse().GetSocks()))
				}
			},
		},
		{
			name:   "env",
			module: consts.ModuleEnv,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Env(f.session, &implantpb.Request{Name: consts.ModuleEnv})
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				if content.GetSpite().GetResponse().GetKv()["COMPUTERNAME"] != "mock-host" {
					t.Fatalf("env kv = %#v, want COMPUTERNAME", content.GetSpite().GetResponse().GetKv())
				}
			},
		},
		{
			name:   "whoami",
			module: consts.ModuleWhoami,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.Whoami(f.session, &implantpb.Request{Name: consts.ModuleWhoami})
			},
			assertContent: func(t *testing.T, content *clientpb.TaskContext) {
				if got := content.GetSpite().GetResponse().GetOutput(); got != `mock-host\operator` {
					t.Fatalf("whoami output = %q", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := len(f.mock.RequestsByName(tc.module))
			task, err := tc.invoke()
			if err != nil {
				t.Fatalf("%s rpc failed: %v", tc.name, err)
			}
			if task == nil || task.TaskId == 0 {
				t.Fatalf("%s task = %#v, want valid task", tc.name, task)
			}

			request := waitModuleRequest(t, f.mock, tc.module, before)
			if tc.assertRequest != nil {
				tc.assertRequest(t, request)
			}

			content := waitTaskFinish(t, f.rpc, f.mock.SessionID, task.TaskId)
			if tc.assertContent != nil {
				tc.assertContent(t, content)
			}
			if tc.assertAfter != nil {
				tc.assertAfter(t)
			}
		})
	}
}

func TestMockImplantInventoryRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	type testCase struct {
		name        string
		module      string
		invoke      func() (*clientpb.Task, error)
		assertTask  func(*testing.T, *clientpb.TaskContext)
		assertState func(*testing.T)
	}

	cases := []testCase{
		{
			name:   "enum drivers",
			module: consts.ModuleEnumDrivers,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.EnumDrivers(f.session, &implantpb.Request{Name: consts.ModuleEnumDrivers})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if len(content.GetSpite().GetEnumDriversResponse().GetDrives()) != 2 {
					t.Fatalf("drive count = %d, want 2", len(content.GetSpite().GetEnumDriversResponse().GetDrives()))
				}
			},
		},
		{
			name:   "service list",
			module: consts.ModuleServiceList,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.ServiceList(f.session, &implantpb.Request{Name: consts.ModuleServiceList})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if len(content.GetSpite().GetServicesResponse().GetServices()) == 0 {
					t.Fatal("service list should not be empty")
				}
			},
		},
		{
			name:   "service query",
			module: consts.ModuleServiceQuery,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.ServiceQuery(f.session, &implantpb.ServiceRequest{
					Type: consts.ModuleServiceQuery,
					Service: &implantpb.ServiceConfig{
						Name: f.lib.ServiceName,
					},
				})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if got := content.GetSpite().GetServiceResponse().GetConfig().GetName(); got != f.lib.ServiceName {
					t.Fatalf("service query name = %q, want %q", got, f.lib.ServiceName)
				}
			},
		},
		{
			name:   "taskschd list",
			module: consts.ModuleTaskSchdList,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.TaskSchdList(f.session, &implantpb.Request{Name: consts.ModuleTaskSchdList})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if len(content.GetSpite().GetSchedulesResponse().GetSchedules()) == 0 {
					t.Fatal("task schedule list should not be empty")
				}
			},
		},
		{
			name:   "taskschd query",
			module: consts.ModuleTaskSchdQuery,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.TaskSchdQuery(f.session, &implantpb.TaskScheduleRequest{
					Type: consts.ModuleTaskSchdQuery,
					Taskschd: &implantpb.TaskSchedule{
						Name: f.lib.ScheduleName,
						Path: `\Malice`,
					},
				})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if got := content.GetSpite().GetScheduleResponse().GetName(); got != f.lib.ScheduleName {
					t.Fatalf("taskschd query name = %q, want %q", got, f.lib.ScheduleName)
				}
			},
		},
		{
			name:   "registry list key",
			module: consts.ModuleRegListKey,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.RegListKey(f.session, &implantpb.RegistryRequest{
					Type: consts.ModuleRegListKey,
					Registry: &implantpb.Registry{
						Hive: f.lib.RegistryHive,
						Path: `SOFTWARE`,
					},
				})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if len(content.GetSpite().GetResponse().GetArray()) == 0 {
					t.Fatal("registry subkeys should not be empty")
				}
			},
		},
		{
			name:   "registry list value",
			module: consts.ModuleRegListValue,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.RegListValue(f.session, &implantpb.RegistryRequest{
					Type: consts.ModuleRegListValue,
					Registry: &implantpb.Registry{
						Hive: f.lib.RegistryHive,
						Path: f.lib.RegistryPath,
					},
				})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if content.GetSpite().GetResponse().GetKv()["InstallPath"] == "" {
					t.Fatalf("registry values = %#v, want InstallPath", content.GetSpite().GetResponse().GetKv())
				}
			},
		},
		{
			name:   "registry query",
			module: consts.ModuleRegQuery,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.RegQuery(f.session, &implantpb.RegistryRequest{
					Type: consts.ModuleRegQuery,
					Registry: &implantpb.Registry{
						Hive: f.lib.RegistryHive,
						Path: f.lib.RegistryPath,
						Key:  "Version",
					},
				})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if content.GetSpite().GetResponse().GetKv()["Version"] != "1.4.2" {
					t.Fatalf("registry query = %#v, want Version=1.4.2", content.GetSpite().GetResponse().GetKv())
				}
			},
		},
		{
			name:   "list module",
			module: consts.ModuleListModule,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.ListModule(f.session, &implantpb.Request{Name: consts.ModuleListModule})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				requireModule(t, content.GetSpite().GetModules().GetModules(), consts.ModuleLs)
				requireModule(t, content.GetSpite().GetModules().GetModules(), consts.ModuleRunas)
				requireModule(t, content.GetSpite().GetModules().GetModules(), consts.ModuleTaskSchdRun)
			},
			assertState: func(t *testing.T) {
				session, err := f.h.GetSession(f.mock.SessionID)
				if err != nil {
					t.Fatalf("GetSession failed: %v", err)
				}
				requireModule(t, session.GetModules(), consts.ModuleLs)
				requireModule(t, session.GetModules(), consts.ModuleRunas)
				requireModule(t, session.GetModules(), consts.ModuleTaskSchdRun)
			},
		},
		{
			name:   "list addon",
			module: consts.ModuleListAddon,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.ListAddon(f.session, &implantpb.Request{Name: consts.ModuleListAddon})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				requireAddon(t, content.GetSpite().GetAddons().GetAddons(), "seatbelt")
			},
			assertState: func(t *testing.T) {
				session, err := f.h.GetSession(f.mock.SessionID)
				if err != nil {
					t.Fatalf("GetSession failed: %v", err)
				}
				requireAddon(t, session.GetAddons(), "seatbelt")
			},
		},
		{
			name:   "list task",
			module: consts.ModuleListTask,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.ListTasks(f.session, &implantpb.Request{Name: consts.ModuleListTask})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if len(content.GetSpite().GetTaskList().GetTasks()) != 2 {
					t.Fatalf("task list count = %d, want 2", len(content.GetSpite().GetTaskList().GetTasks()))
				}
			},
		},
		{
			name:   "query task",
			module: consts.ModuleQueryTask,
			invoke: func() (*clientpb.Task, error) {
				return f.rpc.QueryTask(f.session, &implantpb.TaskCtrl{
					TaskId: 42,
					Op:     consts.ModuleQueryTask,
				})
			},
			assertTask: func(t *testing.T, content *clientpb.TaskContext) {
				if got := content.GetSpite().GetTaskInfo().GetTaskId(); got != 42 {
					t.Fatalf("query task id = %d, want 42", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := len(f.mock.RequestsByName(tc.module))
			task, err := tc.invoke()
			if err != nil {
				t.Fatalf("%s rpc failed: %v", tc.name, err)
			}
			request := waitModuleRequest(t, f.mock, tc.module, before)
			if request.GetSession().GetSessionId() != f.mock.SessionID {
				t.Fatalf("%s session id = %q, want %q", tc.name, request.GetSession().GetSessionId(), f.mock.SessionID)
			}
			content := waitTaskFinish(t, f.rpc, f.mock.SessionID, task.TaskId)
			tc.assertTask(t, content)
			if tc.assertState != nil {
				tc.assertState(t)
			}
		})
	}
}

func TestMockImplantCancelTaskRPCE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	seedBefore := len(f.mock.RequestsByName(consts.ModulePwd))
	seedTask, err := f.rpc.Pwd(f.session, &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("Pwd failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModulePwd, seedBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, seedTask.TaskId)

	cancelBefore := len(f.mock.RequestsByName(consts.ModuleCancelTask))
	cancelTask, err := f.rpc.CancelTask(f.session, &implantpb.TaskCtrl{
		TaskId: seedTask.TaskId,
		Op:     consts.ModuleCancelTask,
	})
	if err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}
	if cancelTask == nil || cancelTask.TaskId == 0 {
		t.Fatalf("cancel task = %#v, want valid task", cancelTask)
	}

	cancelRequest := waitModuleRequest(t, f.mock, consts.ModuleCancelTask, cancelBefore)
	if got := cancelRequest.GetSpite().GetTask().GetTaskId(); got != seedTask.TaskId {
		t.Fatalf("cancel request task id = %d, want %d", got, seedTask.TaskId)
	}
	if got := cancelRequest.GetSpite().GetTask().GetOp(); got != consts.ModuleCancelTask {
		t.Fatalf("cancel request op = %q, want %q", got, consts.ModuleCancelTask)
	}

	cancelContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, cancelTask.TaskId)
	if cancelContent.GetSpite().GetEmpty() == nil {
		t.Fatalf("cancel task content = %#v, want empty response", cancelContent.GetSpite())
	}

	runtimeTask, err := f.h.GetRuntimeTask(f.mock.SessionID, seedTask.TaskId)
	if err != nil {
		t.Fatalf("GetRuntimeTask(target) failed: %v", err)
	}
	if runtimeTask.TaskId != seedTask.TaskId {
		t.Fatalf("runtime task id = %d, want %d", runtimeTask.TaskId, seedTask.TaskId)
	}
	if !runtimeTask.Finished {
		t.Fatalf("runtime task finished = %v, want true", runtimeTask.Finished)
	}

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantMutationRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	setEnvBefore := len(f.mock.RequestsByName(consts.ModuleSetEnv))
	setEnvTask, err := f.rpc.SetEnv(f.session, &implantpb.Request{
		Name: consts.ModuleSetEnv,
		Args: []string{"MALICE_STAGE", "integration"},
	})
	if err != nil {
		t.Fatalf("SetEnv failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleSetEnv, setEnvBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, setEnvTask.TaskId)

	envBefore := len(f.mock.RequestsByName(consts.ModuleEnv))
	envTask, err := f.rpc.Env(f.session, &implantpb.Request{Name: consts.ModuleEnv})
	if err != nil {
		t.Fatalf("Env failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleEnv, envBefore)
	envContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, envTask.TaskId)
	if envContent.GetSpite().GetResponse().GetKv()["MALICE_STAGE"] != "integration" {
		t.Fatalf("env after set = %#v, want MALICE_STAGE=integration", envContent.GetSpite().GetResponse().GetKv())
	}

	unsetBefore := len(f.mock.RequestsByName(consts.ModuleUnsetEnv))
	unsetTask, err := f.rpc.UnsetEnv(f.session, &implantpb.Request{
		Name:  consts.ModuleUnsetEnv,
		Input: "MALICE_STAGE",
	})
	if err != nil {
		t.Fatalf("UnsetEnv failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleUnsetEnv, unsetBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, unsetTask.TaskId)

	envBefore = len(f.mock.RequestsByName(consts.ModuleEnv))
	envTask, err = f.rpc.Env(f.session, &implantpb.Request{Name: consts.ModuleEnv})
	if err != nil {
		t.Fatalf("Env(second) failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleEnv, envBefore)
	envContent = waitTaskFinish(t, f.rpc, f.mock.SessionID, envTask.TaskId)
	if _, ok := envContent.GetSpite().GetResponse().GetKv()["MALICE_STAGE"]; ok {
		t.Fatalf("env after unset = %#v, want MALICE_STAGE removed", envContent.GetSpite().GetResponse().GetKv())
	}

	regAddBefore := len(f.mock.RequestsByName(consts.ModuleRegAdd))
	regAddTask, err := f.rpc.RegAdd(f.session, &implantpb.RegistryWriteRequest{
		Hive:        f.lib.RegistryHive,
		Path:        f.lib.RegistryPath,
		Key:         "BeaconInterval",
		StringValue: "45s",
		Regtype:     1,
	})
	if err != nil {
		t.Fatalf("RegAdd failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleRegAdd, regAddBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, regAddTask.TaskId)

	regQueryBefore := len(f.mock.RequestsByName(consts.ModuleRegQuery))
	regQueryTask, err := f.rpc.RegQuery(f.session, &implantpb.RegistryRequest{
		Type: consts.ModuleRegQuery,
		Registry: &implantpb.Registry{
			Hive: f.lib.RegistryHive,
			Path: f.lib.RegistryPath,
			Key:  "BeaconInterval",
		},
	})
	if err != nil {
		t.Fatalf("RegQuery failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleRegQuery, regQueryBefore)
	regQueryContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, regQueryTask.TaskId)
	if regQueryContent.GetSpite().GetResponse().GetKv()["BeaconInterval"] != "45s" {
		t.Fatalf("registry after add = %#v, want BeaconInterval=45s", regQueryContent.GetSpite().GetResponse().GetKv())
	}

	regDeleteBefore := len(f.mock.RequestsByName(consts.ModuleRegDelete))
	regDeleteTask, err := f.rpc.RegDelete(f.session, &implantpb.RegistryRequest{
		Type: consts.ModuleRegDelete,
		Registry: &implantpb.Registry{
			Hive: f.lib.RegistryHive,
			Path: f.lib.RegistryPath,
			Key:  "BeaconInterval",
		},
	})
	if err != nil {
		t.Fatalf("RegDelete failed: %v", err)
	}
	regDeleteRequest := waitModuleRequest(t, f.mock, consts.ModuleRegDelete, regDeleteBefore)
	if got := regDeleteRequest.GetSpite().GetRegistryRequest().GetKey(); got != "BeaconInterval" {
		t.Fatalf("reg delete key = %q, want BeaconInterval", got)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, regDeleteTask.TaskId)

	regQueryBefore = len(f.mock.RequestsByName(consts.ModuleRegQuery))
	regQueryTask, err = f.rpc.RegQuery(f.session, &implantpb.RegistryRequest{
		Type: consts.ModuleRegQuery,
		Registry: &implantpb.Registry{
			Hive: f.lib.RegistryHive,
			Path: f.lib.RegistryPath,
			Key:  "BeaconInterval",
		},
	})
	if err != nil {
		t.Fatalf("RegQuery(second) failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleRegQuery, regQueryBefore)
	regQueryContent = waitTaskFinish(t, f.rpc, f.mock.SessionID, regQueryTask.TaskId)
	if got := regQueryContent.GetSpite().GetResponse().GetKv()["BeaconInterval"]; got != "" {
		t.Fatalf("registry after delete = %#v, want BeaconInterval removed", regQueryContent.GetSpite().GetResponse().GetKv())
	}

	serviceCreateBefore := len(f.mock.RequestsByName(consts.ModuleServiceCreate))
	serviceCreateTask, err := f.rpc.ServiceCreate(f.session, &implantpb.ServiceRequest{
		Type: consts.ModuleServiceCreate,
		Service: &implantpb.ServiceConfig{
			Name:           "MaliceAgent",
			DisplayName:    "Malice Agent",
			ExecutablePath: `C:\Program Files\Malice\agent.exe`,
			StartType:      2,
			ErrorControl:   1,
			AccountName:    `LocalSystem`,
		},
	})
	if err != nil {
		t.Fatalf("ServiceCreate failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleServiceCreate, serviceCreateBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, serviceCreateTask.TaskId)

	serviceQueryBefore := len(f.mock.RequestsByName(consts.ModuleServiceQuery))
	serviceQueryTask, err := f.rpc.ServiceQuery(f.session, &implantpb.ServiceRequest{
		Type:    consts.ModuleServiceQuery,
		Service: &implantpb.ServiceConfig{Name: "MaliceAgent"},
	})
	if err != nil {
		t.Fatalf("ServiceQuery failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleServiceQuery, serviceQueryBefore)
	serviceQueryContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, serviceQueryTask.TaskId)
	if got := serviceQueryContent.GetSpite().GetServiceResponse().GetConfig().GetExecutablePath(); got != `C:\Program Files\Malice\agent.exe` {
		t.Fatalf("service executable path = %q", got)
	}

	taskCreateBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdCreate))
	taskCreateTask, err := f.rpc.TaskSchdCreate(f.session, &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdCreate,
		Taskschd: &implantpb.TaskSchedule{
			Name:           "MaliceWeeklySweep",
			Path:           `\Malice`,
			ExecutablePath: `C:\Program Files\Malice\sweep.exe`,
			TriggerType:    3,
			StartBoundary:  "2026-03-21T09:00:00",
		},
	})
	if err != nil {
		t.Fatalf("TaskSchdCreate failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleTaskSchdCreate, taskCreateBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, taskCreateTask.TaskId)

	taskQueryBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdQuery))
	taskQueryTask, err := f.rpc.TaskSchdQuery(f.session, &implantpb.TaskScheduleRequest{
		Type:     consts.ModuleTaskSchdQuery,
		Taskschd: &implantpb.TaskSchedule{Name: "MaliceWeeklySweep", Path: `\Malice`},
	})
	if err != nil {
		t.Fatalf("TaskSchdQuery failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleTaskSchdQuery, taskQueryBefore)
	taskQueryContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, taskQueryTask.TaskId)
	if !taskQueryContent.GetSpite().GetScheduleResponse().GetEnabled() {
		t.Fatal("created schedule should be enabled")
	}

	loadModuleBefore := len(f.mock.RequestsByName(consts.ModuleLoadModule))
	loadModuleTask, err := f.rpc.LoadModule(f.session, &implantpb.LoadModule{
		Bundle: "sharpkatz",
		Bin:    []byte("bundle"),
	})
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleLoadModule, loadModuleBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, loadModuleTask.TaskId)

	session, err := f.h.GetSession(f.mock.SessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	requireModule(t, session.GetModules(), "sharpkatz")

	loadAddonBefore := len(f.mock.RequestsByName(consts.ModuleLoadAddon))
	loadAddonTask, err := f.rpc.LoadAddon(f.session, &implantpb.LoadAddon{
		Name:   "recon-kit",
		Type:   "bof",
		Depend: consts.ModuleExecute,
		Bin:    []byte("addon"),
	})
	if err != nil {
		t.Fatalf("LoadAddon failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleLoadAddon, loadAddonBefore)
	waitTaskFinish(t, f.rpc, f.mock.SessionID, loadAddonTask.TaskId)

	session, err = f.h.GetSession(f.mock.SessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	requireAddon(t, session.GetAddons(), "recon-kit")

	execAddonBefore := len(f.mock.RequestsByName(consts.ModuleExecuteAddon))
	execAddonTask, err := f.rpc.ExecuteAddon(f.session, &implantpb.ExecuteAddon{
		Addon: "recon-kit",
		ExecuteBinary: &implantpb.ExecuteBinary{
			Name:   "recon-kit",
			Bin:    []byte("addon"),
			Output: true,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteAddon failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleExecuteAddon, execAddonBefore)
	execAddonContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, execAddonTask.TaskId)
	if got := string(execAddonContent.GetSpite().GetBinaryResponse().GetData()); got != "addon:recon-kit:ok" {
		t.Fatalf("execute addon result = %q", got)
	}

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantFilesystemMutationRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	scratchDir := f.lib.WorkDir + `\ops`
	markerPath := scratchDir + `\marker.txt`
	copiedPath := scratchDir + `\notes-copy.txt`
	movedPath := scratchDir + `\notes-moved.txt`

	mkdirBefore := len(f.mock.RequestsByName(consts.ModuleMkdir))
	mkdirTask, err := f.rpc.Mkdir(f.session, &implantpb.Request{
		Name:  consts.ModuleMkdir,
		Input: scratchDir,
	})
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	mkdirRequest := waitModuleRequest(t, f.mock, consts.ModuleMkdir, mkdirBefore)
	if got := mkdirRequest.GetSpite().GetRequest().GetInput(); got != scratchDir {
		t.Fatalf("mkdir input = %q, want %q", got, scratchDir)
	}
	mkdirContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, mkdirTask.TaskId)
	if mkdirContent.GetSpite().GetEmpty() == nil {
		t.Fatalf("mkdir response = %#v, want empty response", mkdirContent.GetSpite())
	}

	cdBefore := len(f.mock.RequestsByName(consts.ModuleCd))
	cdTask, err := f.rpc.Cd(f.session, &implantpb.Request{
		Name:  consts.ModuleCd,
		Input: scratchDir,
	})
	if err != nil {
		t.Fatalf("Cd failed: %v", err)
	}
	cdRequest := waitModuleRequest(t, f.mock, consts.ModuleCd, cdBefore)
	if got := cdRequest.GetSpite().GetRequest().GetInput(); got != scratchDir {
		t.Fatalf("cd input = %q, want %q", got, scratchDir)
	}
	cdContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, cdTask.TaskId)
	if got := cdContent.GetSpite().GetResponse().GetOutput(); got != normalizePath(scratchDir) {
		t.Fatalf("cd output = %q, want %q", got, normalizePath(scratchDir))
	}

	pwdBefore := len(f.mock.RequestsByName(consts.ModulePwd))
	pwdTask, err := f.rpc.Pwd(f.session, &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("Pwd failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModulePwd, pwdBefore)
	pwdContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, pwdTask.TaskId)
	if got := pwdContent.GetSpite().GetResponse().GetOutput(); got != normalizePath(scratchDir) {
		t.Fatalf("pwd after cd = %q, want %q", got, normalizePath(scratchDir))
	}

	touchBefore := len(f.mock.RequestsByName(consts.ModuleTouch))
	touchTask, err := f.rpc.Touch(f.session, &implantpb.Request{
		Name:  consts.ModuleTouch,
		Input: markerPath,
	})
	if err != nil {
		t.Fatalf("Touch failed: %v", err)
	}
	touchRequest := waitModuleRequest(t, f.mock, consts.ModuleTouch, touchBefore)
	if got := touchRequest.GetSpite().GetRequest().GetInput(); got != markerPath {
		t.Fatalf("touch input = %q, want %q", got, markerPath)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, touchTask.TaskId)

	cpBefore := len(f.mock.RequestsByName(consts.ModuleCp))
	cpTask, err := f.rpc.Cp(f.session, &implantpb.Request{
		Name: consts.ModuleCp,
		Args: []string{f.lib.NotesPath, copiedPath},
	})
	if err != nil {
		t.Fatalf("Cp failed: %v", err)
	}
	cpRequest := waitModuleRequest(t, f.mock, consts.ModuleCp, cpBefore)
	if got := cpRequest.GetSpite().GetRequest().GetArgs(); len(got) != 2 || got[0] != f.lib.NotesPath || got[1] != copiedPath {
		t.Fatalf("cp args = %#v, want [%q %q]", got, f.lib.NotesPath, copiedPath)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, cpTask.TaskId)

	catBefore := len(f.mock.RequestsByName(consts.ModuleCat))
	catTask, err := f.rpc.Cat(f.session, &implantpb.Request{
		Name:  consts.ModuleCat,
		Input: copiedPath,
	})
	if err != nil {
		t.Fatalf("Cat failed: %v", err)
	}
	catRequest := waitModuleRequest(t, f.mock, consts.ModuleCat, catBefore)
	if got := catRequest.GetSpite().GetRequest().GetInput(); got != copiedPath {
		t.Fatalf("cat input = %q, want %q", got, copiedPath)
	}
	catContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, catTask.TaskId)
	if got := string(catContent.GetSpite().GetBinaryResponse().GetData()); !strings.Contains(got, "rpc coverage") {
		t.Fatalf("cat copied content = %q, want copied mock payload", got)
	}

	mvBefore := len(f.mock.RequestsByName(consts.ModuleMv))
	mvTask, err := f.rpc.Mv(f.session, &implantpb.Request{
		Name: consts.ModuleMv,
		Args: []string{copiedPath, movedPath},
	})
	if err != nil {
		t.Fatalf("Mv failed: %v", err)
	}
	mvRequest := waitModuleRequest(t, f.mock, consts.ModuleMv, mvBefore)
	if got := mvRequest.GetSpite().GetRequest().GetArgs(); len(got) != 2 || got[0] != copiedPath || got[1] != movedPath {
		t.Fatalf("mv args = %#v, want [%q %q]", got, copiedPath, movedPath)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, mvTask.TaskId)

	lsBefore := len(f.mock.RequestsByName(consts.ModuleLs))
	lsTask, err := f.rpc.Ls(f.session, &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: scratchDir,
	})
	if err != nil {
		t.Fatalf("Ls failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleLs, lsBefore)
	lsContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, lsTask.TaskId)
	requireFileInfo(t, lsContent.GetSpite().GetLsResponse().GetFiles(), "marker.txt")
	requireFileInfo(t, lsContent.GetSpite().GetLsResponse().GetFiles(), "notes-moved.txt")
	requireNoFileInfo(t, lsContent.GetSpite().GetLsResponse().GetFiles(), "notes-copy.txt")

	rmBefore := len(f.mock.RequestsByName(consts.ModuleRm))
	rmTask, err := f.rpc.Rm(f.session, &implantpb.Request{
		Name:  consts.ModuleRm,
		Input: movedPath,
	})
	if err != nil {
		t.Fatalf("Rm failed: %v", err)
	}
	rmRequest := waitModuleRequest(t, f.mock, consts.ModuleRm, rmBefore)
	if got := rmRequest.GetSpite().GetRequest().GetInput(); got != movedPath {
		t.Fatalf("rm input = %q, want %q", got, movedPath)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, rmTask.TaskId)

	lsBefore = len(f.mock.RequestsByName(consts.ModuleLs))
	lsTask, err = f.rpc.Ls(f.session, &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: scratchDir,
	})
	if err != nil {
		t.Fatalf("Ls(second) failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleLs, lsBefore)
	lsContent = waitTaskFinish(t, f.rpc, f.mock.SessionID, lsTask.TaskId)
	requireFileInfo(t, lsContent.GetSpite().GetLsResponse().GetFiles(), "marker.txt")
	requireNoFileInfo(t, lsContent.GetSpite().GetLsResponse().GetFiles(), "notes-moved.txt")

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantFileTransferRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	uploadedPath := fileutils.RemoteJoin(f.lib.WorkDir, "uploaded.txt")
	uploadBody := []byte("uploaded from mock implant e2e")

	uploadBefore := len(f.mock.RequestsByName(consts.ModuleUpload))
	uploadTask, err := f.rpc.Upload(f.session, &implantpb.UploadRequest{
		Name:   "local.txt",
		Target: uploadedPath,
		Priv:   0o640,
		Data:   uploadBody,
		Hidden: true,
	})
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	uploadRequest := waitModuleRequest(t, f.mock, consts.ModuleUpload, uploadBefore)
	if got := uploadRequest.GetSpite().GetUploadRequest().GetTarget(); got != uploadedPath {
		t.Fatalf("upload target = %q, want %q", got, uploadedPath)
	}
	if got := uploadRequest.GetSpite().GetUploadRequest().GetPriv(); got != 0o640 {
		t.Fatalf("upload priv = %d, want 0640", got)
	}
	if !uploadRequest.GetSpite().GetUploadRequest().GetHidden() {
		t.Fatal("upload hidden flag should be true")
	}
	if got := string(uploadRequest.GetSpite().GetUploadRequest().GetData()); got != string(uploadBody) {
		t.Fatalf("upload data = %q, want %q", got, string(uploadBody))
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, uploadTask.TaskId)

	catBefore := len(f.mock.RequestsByName(consts.ModuleCat))
	catTask, err := f.rpc.Cat(f.session, &implantpb.Request{Name: consts.ModuleCat, Input: uploadedPath})
	if err != nil {
		t.Fatalf("Cat(uploaded) failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleCat, catBefore)
	catContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, catTask.TaskId)
	if got := string(catContent.GetSpite().GetBinaryResponse().GetData()); got != string(uploadBody) {
		t.Fatalf("uploaded file content = %q, want %q", got, string(uploadBody))
	}

	uploadContexts, err := f.rpc.GetContexts(context.Background(), &clientpb.Context{
		Type: consts.ContextUpload,
		Task: &clientpb.Task{SessionId: f.mock.SessionID, TaskId: uploadTask.TaskId},
	})
	if err != nil {
		t.Fatalf("GetContexts(upload) failed: %v", err)
	}
	if len(uploadContexts.GetContexts()) != 1 {
		t.Fatalf("upload contexts = %d, want 1", len(uploadContexts.GetContexts()))
	}
	uploadCtx, err := output.ToContext[*output.UploadContext](uploadContexts.GetContexts()[0])
	if err != nil {
		t.Fatalf("parse upload context failed: %v", err)
	}
	if uploadCtx.TargetPath != uploadedPath {
		t.Fatalf("upload context target = %q, want %q", uploadCtx.TargetPath, uploadedPath)
	}

	downloadBefore := len(f.mock.RequestsByName(consts.ModuleDownload))
	downloadTask, err := f.rpc.Download(f.session, &implantpb.DownloadRequest{
		Path: f.lib.NotesPath,
		Name: fileutils.RemoteBase(f.lib.NotesPath),
	})
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	firstDownloadRequest := waitModuleRequest(t, f.mock, consts.ModuleDownload, downloadBefore)
	if got := firstDownloadRequest.GetSpite().GetDownloadRequest().GetPath(); got != f.lib.NotesPath {
		t.Fatalf("download path = %q, want %q", got, f.lib.NotesPath)
	}
	if got := firstDownloadRequest.GetSpite().GetDownloadRequest().GetCur(); got != 0 {
		t.Fatalf("download initial cur = %d, want 0", got)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, downloadTask.TaskId)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(f.mock.RequestsByName(consts.ModuleDownload)) >= downloadBefore+2 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	downloadRequests := f.mock.RequestsByName(consts.ModuleDownload)
	if len(downloadRequests) < downloadBefore+2 {
		allRequests := f.mock.Requests()
		names := make([]string, 0, len(allRequests))
		for _, request := range allRequests {
			name := ""
			if request.GetSpite() != nil {
				name = request.GetSpite().GetName()
			}
			names = append(names, name)
		}
		t.Fatalf("download request count = %d, want >= %d; all request names=%v", len(downloadRequests), downloadBefore+2, names)
	}
	lastDownloadRequest := downloadRequests[len(downloadRequests)-1]
	if got := lastDownloadRequest.GetSpite().GetDownloadRequest().GetCur(); got != 1 {
		t.Fatalf("download follow-up cur = %d, want 1", got)
	}

	downloadContexts, err := f.rpc.GetContexts(context.Background(), &clientpb.Context{
		Type: consts.ContextDownload,
		Task: &clientpb.Task{SessionId: f.mock.SessionID, TaskId: downloadTask.TaskId},
	})
	if err != nil {
		t.Fatalf("GetContexts(download) failed: %v", err)
	}
	if len(downloadContexts.GetContexts()) != 1 {
		t.Fatalf("download contexts = %d, want 1", len(downloadContexts.GetContexts()))
	}
	downloadCtx, err := output.ToContext[*output.DownloadContext](downloadContexts.GetContexts()[0])
	if err != nil {
		t.Fatalf("parse download context failed: %v", err)
	}
	if downloadCtx.TargetPath != f.lib.NotesPath {
		t.Fatalf("download context target = %q, want %q", downloadCtx.TargetPath, f.lib.NotesPath)
	}
	data, err := os.ReadFile(downloadCtx.FilePath)
	if err != nil {
		t.Fatalf("read downloaded file failed: %v", err)
	}
	if !strings.Contains(string(data), "rpc coverage") {
		t.Fatalf("downloaded file content = %q, want mock fixture payload", string(data))
	}

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantServiceLifecycleRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	serviceName := "MaliceAgentLifecycle"

	createBefore := len(f.mock.RequestsByName(consts.ModuleServiceCreate))
	createTask, err := f.rpc.ServiceCreate(f.session, &implantpb.ServiceRequest{
		Type: consts.ModuleServiceCreate,
		Service: &implantpb.ServiceConfig{
			Name:           serviceName,
			DisplayName:    "Malice Agent Lifecycle",
			ExecutablePath: `C:\Program Files\Malice\agent-lifecycle.exe`,
			StartType:      2,
			ErrorControl:   1,
			AccountName:    `LocalSystem`,
		},
	})
	if err != nil {
		t.Fatalf("ServiceCreate failed: %v", err)
	}
	createRequest := waitModuleRequest(t, f.mock, consts.ModuleServiceCreate, createBefore)
	if got := createRequest.GetSpite().GetServiceRequest().GetName(); got != serviceName {
		t.Fatalf("service create name = %q, want %q", got, serviceName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, createTask.TaskId)

	queryService := func() *implantpb.Service {
		queryBefore := len(f.mock.RequestsByName(consts.ModuleServiceQuery))
		queryTask, err := f.rpc.ServiceQuery(f.session, &implantpb.ServiceRequest{
			Type:    consts.ModuleServiceQuery,
			Service: &implantpb.ServiceConfig{Name: serviceName},
		})
		if err != nil {
			t.Fatalf("ServiceQuery failed: %v", err)
		}
		queryRequest := waitModuleRequest(t, f.mock, consts.ModuleServiceQuery, queryBefore)
		if got := queryRequest.GetSpite().GetServiceRequest().GetName(); got != serviceName {
			t.Fatalf("service query name = %q, want %q", got, serviceName)
		}
		queryContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, queryTask.TaskId)
		return queryContent.GetSpite().GetServiceResponse()
	}

	service := queryService()
	if got := service.GetConfig().GetExecutablePath(); got != `C:\Program Files\Malice\agent-lifecycle.exe` {
		t.Fatalf("created service path = %q", got)
	}
	if got := service.GetStatus().GetCurrentState(); got != 1 {
		t.Fatalf("created service state = %d, want 1", got)
	}

	startBefore := len(f.mock.RequestsByName(consts.ModuleServiceStart))
	startTask, err := f.rpc.ServiceStart(f.session, &implantpb.ServiceRequest{
		Type:    consts.ModuleServiceStart,
		Service: &implantpb.ServiceConfig{Name: serviceName},
	})
	if err != nil {
		t.Fatalf("ServiceStart failed: %v", err)
	}
	startRequest := waitModuleRequest(t, f.mock, consts.ModuleServiceStart, startBefore)
	if got := startRequest.GetSpite().GetServiceRequest().GetName(); got != serviceName {
		t.Fatalf("service start name = %q, want %q", got, serviceName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, startTask.TaskId)

	service = queryService()
	if got := service.GetStatus().GetCurrentState(); got != 4 {
		t.Fatalf("running service state = %d, want 4", got)
	}
	if got := service.GetStatus().GetProcessId(); got != 4242 {
		t.Fatalf("running service pid = %d, want 4242", got)
	}

	stopBefore := len(f.mock.RequestsByName(consts.ModuleServiceStop))
	stopTask, err := f.rpc.ServiceStop(f.session, &implantpb.ServiceRequest{
		Type:    consts.ModuleServiceStop,
		Service: &implantpb.ServiceConfig{Name: serviceName},
	})
	if err != nil {
		t.Fatalf("ServiceStop failed: %v", err)
	}
	stopRequest := waitModuleRequest(t, f.mock, consts.ModuleServiceStop, stopBefore)
	if got := stopRequest.GetSpite().GetServiceRequest().GetName(); got != serviceName {
		t.Fatalf("service stop name = %q, want %q", got, serviceName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, stopTask.TaskId)

	service = queryService()
	if got := service.GetStatus().GetCurrentState(); got != 1 {
		t.Fatalf("stopped service state = %d, want 1", got)
	}
	if got := service.GetStatus().GetProcessId(); got != 0 {
		t.Fatalf("stopped service pid = %d, want 0", got)
	}

	deleteBefore := len(f.mock.RequestsByName(consts.ModuleServiceDelete))
	deleteTask, err := f.rpc.ServiceDelete(f.session, &implantpb.ServiceRequest{
		Type:    consts.ModuleServiceDelete,
		Service: &implantpb.ServiceConfig{Name: serviceName},
	})
	if err != nil {
		t.Fatalf("ServiceDelete failed: %v", err)
	}
	deleteRequest := waitModuleRequest(t, f.mock, consts.ModuleServiceDelete, deleteBefore)
	if got := deleteRequest.GetSpite().GetServiceRequest().GetName(); got != serviceName {
		t.Fatalf("service delete name = %q, want %q", got, serviceName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, deleteTask.TaskId)

	listBefore := len(f.mock.RequestsByName(consts.ModuleServiceList))
	listTask, err := f.rpc.ServiceList(f.session, &implantpb.Request{Name: consts.ModuleServiceList})
	if err != nil {
		t.Fatalf("ServiceList failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleServiceList, listBefore)
	listContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, listTask.TaskId)
	if serviceByName(listContent.GetSpite().GetServicesResponse().GetServices(), serviceName) != nil {
		t.Fatalf("service list still contains %q after delete", serviceName)
	}

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantTaskScheduleLifecycleRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	scheduleName := "MaliceHourlySweep"

	createBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdCreate))
	createTask, err := f.rpc.TaskSchdCreate(f.session, &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdCreate,
		Taskschd: &implantpb.TaskSchedule{
			Name:           scheduleName,
			Path:           `\Malice`,
			ExecutablePath: `C:\Program Files\Malice\hourly-sweep.exe`,
			TriggerType:    1,
			StartBoundary:  "2026-03-14T10:00:00",
		},
	})
	if err != nil {
		t.Fatalf("TaskSchdCreate failed: %v", err)
	}
	createRequest := waitModuleRequest(t, f.mock, consts.ModuleTaskSchdCreate, createBefore)
	if got := createRequest.GetSpite().GetScheduleRequest().GetName(); got != scheduleName {
		t.Fatalf("taskschd create name = %q, want %q", got, scheduleName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, createTask.TaskId)

	querySchedule := func() *implantpb.TaskSchedule {
		queryBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdQuery))
		queryTask, err := f.rpc.TaskSchdQuery(f.session, &implantpb.TaskScheduleRequest{
			Type:     consts.ModuleTaskSchdQuery,
			Taskschd: &implantpb.TaskSchedule{Name: scheduleName, Path: `\Malice`},
		})
		if err != nil {
			t.Fatalf("TaskSchdQuery failed: %v", err)
		}
		queryRequest := waitModuleRequest(t, f.mock, consts.ModuleTaskSchdQuery, queryBefore)
		if got := queryRequest.GetSpite().GetScheduleRequest().GetName(); got != scheduleName {
			t.Fatalf("taskschd query name = %q, want %q", got, scheduleName)
		}
		queryContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, queryTask.TaskId)
		return queryContent.GetSpite().GetScheduleResponse()
	}

	schedule := querySchedule()
	if !schedule.GetEnabled() {
		t.Fatal("created task schedule should be enabled")
	}
	if got := schedule.GetNextRunTime(); got != "2026-03-14T10:00:00" {
		t.Fatalf("created schedule next run = %q, want start boundary", got)
	}

	stopBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdStop))
	stopTask, err := f.rpc.TaskSchdStop(f.session, &implantpb.TaskScheduleRequest{
		Type:     consts.ModuleTaskSchdStop,
		Taskschd: &implantpb.TaskSchedule{Name: scheduleName, Path: `\Malice`},
	})
	if err != nil {
		t.Fatalf("TaskSchdStop failed: %v", err)
	}
	stopRequest := waitModuleRequest(t, f.mock, consts.ModuleTaskSchdStop, stopBefore)
	if got := stopRequest.GetSpite().GetScheduleRequest().GetName(); got != scheduleName {
		t.Fatalf("taskschd stop name = %q, want %q", got, scheduleName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, stopTask.TaskId)

	schedule = querySchedule()
	if schedule.GetEnabled() {
		t.Fatal("stopped schedule should be disabled")
	}

	startBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdStart))
	startTask, err := f.rpc.TaskSchdStart(f.session, &implantpb.TaskScheduleRequest{
		Type:     consts.ModuleTaskSchdStart,
		Taskschd: &implantpb.TaskSchedule{Name: scheduleName, Path: `\Malice`},
	})
	if err != nil {
		t.Fatalf("TaskSchdStart failed: %v", err)
	}
	startRequest := waitModuleRequest(t, f.mock, consts.ModuleTaskSchdStart, startBefore)
	if got := startRequest.GetSpite().GetScheduleRequest().GetName(); got != scheduleName {
		t.Fatalf("taskschd start name = %q, want %q", got, scheduleName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, startTask.TaskId)

	schedule = querySchedule()
	if !schedule.GetEnabled() {
		t.Fatal("started schedule should be enabled")
	}

	runBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdRun))
	runTask, err := f.rpc.TaskSchdRun(f.session, &implantpb.TaskScheduleRequest{
		Type:     consts.ModuleTaskSchdRun,
		Taskschd: &implantpb.TaskSchedule{Name: scheduleName, Path: `\Malice`},
	})
	if err != nil {
		t.Fatalf("TaskSchdRun failed: %v", err)
	}
	runRequest := waitModuleRequest(t, f.mock, consts.ModuleTaskSchdRun, runBefore)
	if got := runRequest.GetSpite().GetScheduleRequest().GetName(); got != scheduleName {
		t.Fatalf("taskschd run name = %q, want %q", got, scheduleName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, runTask.TaskId)

	schedule = querySchedule()
	if got := schedule.GetLastRunTime(); got != "2026-03-14T12:34:56" {
		t.Fatalf("schedule last run = %q, want updated run marker", got)
	}

	deleteBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdDelete))
	deleteTask, err := f.rpc.TaskSchdDelete(f.session, &implantpb.TaskScheduleRequest{
		Type:     consts.ModuleTaskSchdDelete,
		Taskschd: &implantpb.TaskSchedule{Name: scheduleName, Path: `\Malice`},
	})
	if err != nil {
		t.Fatalf("TaskSchdDelete failed: %v", err)
	}
	deleteRequest := waitModuleRequest(t, f.mock, consts.ModuleTaskSchdDelete, deleteBefore)
	if got := deleteRequest.GetSpite().GetScheduleRequest().GetName(); got != scheduleName {
		t.Fatalf("taskschd delete name = %q, want %q", got, scheduleName)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, deleteTask.TaskId)

	listBefore := len(f.mock.RequestsByName(consts.ModuleTaskSchdList))
	listTask, err := f.rpc.TaskSchdList(f.session, &implantpb.Request{Name: consts.ModuleTaskSchdList})
	if err != nil {
		t.Fatalf("TaskSchdList failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModuleTaskSchdList, listBefore)
	listContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, listTask.TaskId)
	if scheduleByName(listContent.GetSpite().GetSchedulesResponse().GetSchedules(), scheduleName) != nil {
		t.Fatalf("task schedule list still contains %q after delete", scheduleName)
	}

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantControlAndExecutionRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	enableBefore := len(f.mock.RequestsByName(consts.ModuleKeepalive))
	enableTask, err := f.rpc.Keepalive(f.session, &implantpb.CommonBody{
		BoolArray: []bool{true},
	})
	if err != nil {
		t.Fatalf("Keepalive(enable) failed: %v", err)
	}
	enableRequest := waitModuleRequest(t, f.mock, consts.ModuleKeepalive, enableBefore)
	if got := enableRequest.GetSpite().GetCommon().GetBoolArray(); len(got) != 1 || !got[0] {
		t.Fatalf("keepalive enable request = %#v, want [true]", got)
	}
	enableContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, enableTask.TaskId)
	if got := enableContent.GetSpite().GetCommon().GetBoolArray(); len(got) != 1 || !got[0] {
		t.Fatalf("keepalive enable response = %#v, want [true]", got)
	}
	sessionRuntime, err := core.Sessions.Get(f.mock.SessionID)
	if err != nil {
		t.Fatalf("core.Sessions.Get failed: %v", err)
	}
	if !sessionRuntime.IsKeepaliveEnabled() {
		t.Fatal("keepalive runtime state should be enabled")
	}

	disableBefore := len(f.mock.RequestsByName(consts.ModuleKeepalive))
	disableTask, err := f.rpc.Keepalive(f.session, &implantpb.CommonBody{
		BoolArray: []bool{false},
	})
	if err != nil {
		t.Fatalf("Keepalive(disable) failed: %v", err)
	}
	disableRequest := waitModuleRequest(t, f.mock, consts.ModuleKeepalive, disableBefore)
	if got := disableRequest.GetSpite().GetCommon().GetBoolArray(); len(got) != 1 || got[0] {
		t.Fatalf("keepalive disable request = %#v, want [false]", got)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, disableTask.TaskId)
	if sessionRuntime.IsKeepaliveEnabled() {
		t.Fatal("keepalive runtime state should be disabled")
	}

	switchBefore := len(f.mock.RequestsByName(consts.ModuleSwitch))
	switchTask, err := f.rpc.Switch(f.session, &implantpb.Switch{
		Targets: []*implantpb.Target{
			{Address: "edge-a.example:443", Protocol: "tcp"},
			{Address: "edge-b.example:443", Protocol: "tcp"},
		},
	})
	if err != nil {
		t.Fatalf("Switch failed: %v", err)
	}
	switchRequest := waitModuleRequest(t, f.mock, consts.ModuleSwitch, switchBefore)
	if got := switchRequest.GetSpite().GetSwitch().GetTargets(); len(got) != 2 || got[0].GetAddress() != "edge-a.example:443" {
		t.Fatalf("switch targets = %#v", got)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, switchTask.TaskId)

	clearBefore := len(f.mock.RequestsByName(consts.ModuleClear))
	clearTask, err := f.rpc.Clear(f.session, &implantpb.Request{Name: consts.ModuleClear})
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	clearRequest := waitModuleRequest(t, f.mock, consts.ModuleClear, clearBefore)
	if got := clearRequest.GetSpite().GetRequest().GetName(); got != consts.ModuleClear {
		t.Fatalf("clear name = %q, want %q", got, consts.ModuleClear)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, clearTask.TaskId)

	execBefore := len(f.mock.RequestsByName(consts.ModuleExecute))
	execTask, err := f.rpc.Execute(f.session, &implantpb.ExecRequest{
		Path:   "cmd.exe",
		Args:   []string{"/c", "hostname"},
		Output: true,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	execRequest := waitModuleRequest(t, f.mock, consts.ModuleExecute, execBefore)
	if got := execRequest.GetSpite().GetExecRequest().GetPath(); got != "cmd.exe" {
		t.Fatalf("execute path = %q, want cmd.exe", got)
	}
	if execRequest.GetSpite().GetExecRequest().GetRealtime() {
		t.Fatal("execute request should preserve realtime=false")
	}
	execContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, execTask.TaskId)
	if got := strings.TrimSpace(string(execContent.GetSpite().GetExecResponse().GetStdout())); got != "mock-host" {
		t.Fatalf("execute stdout = %q, want mock-host", got)
	}
	if !execContent.GetSpite().GetExecResponse().GetEnd() {
		t.Fatal("execute response should end the stream")
	}

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}

func TestMockImplantSystemActionRPCsE2E(t *testing.T) {
	f := newMockRPCFixture(t)

	curlBefore := len(f.mock.RequestsByName(consts.ModuleRequest))
	curlTask, err := f.rpc.Curl(f.session, &implantpb.CurlRequest{
		Method:   "POST",
		Url:      "https://api.example.test/checkin",
		Timeout:  15,
		Body:     []byte(`{"op":"ping"}`),
		Header:   map[string]string{"X-Trace": "mock-001"},
		Hostname: "edge.example.test",
	})
	if err != nil {
		t.Fatalf("Curl failed: %v", err)
	}
	curlRequest := waitModuleRequest(t, f.mock, consts.ModuleRequest, curlBefore)
	if got := curlRequest.GetSpite().GetCurlRequest().GetUrl(); got != "https://api.example.test/checkin" {
		t.Fatalf("curl url = %q", got)
	}
	if got := curlRequest.GetSpite().GetCurlRequest().GetMethod(); got != "POST" {
		t.Fatalf("curl method = %q", got)
	}
	curlContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, curlTask.TaskId)
	if got := string(curlContent.GetSpite().GetBinaryResponse().GetData()); !strings.Contains(got, "mock-curl POST https://api.example.test/checkin") {
		t.Fatalf("curl response = %q", got)
	}

	wmiQueryBefore := len(f.mock.RequestsByName(consts.ModuleWmiQuery))
	wmiQueryTask, err := f.rpc.WmiQuery(f.session, &implantpb.WmiQueryRequest{
		Namespace: `ROOT\CIMV2`,
		Args:      []string{"SELECT", "Caption", "FROM", "Win32_OperatingSystem"},
	})
	if err != nil {
		t.Fatalf("WmiQuery failed: %v", err)
	}
	wmiQueryRequest := waitModuleRequest(t, f.mock, consts.ModuleWmiQuery, wmiQueryBefore)
	if got := wmiQueryRequest.GetSpite().GetWmiRequest().GetNamespace(); got != `ROOT\CIMV2` {
		t.Fatalf("wmi query namespace = %q", got)
	}
	wmiQueryContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, wmiQueryTask.TaskId)
	if got := wmiQueryContent.GetSpite().GetResponse().GetOutput(); !strings.Contains(got, "Win32_OperatingSystem") {
		t.Fatalf("wmi query response = %q", got)
	}

	wmiExecBefore := len(f.mock.RequestsByName(consts.ModuleWmiExec))
	wmiExecTask, err := f.rpc.WmiExecute(f.session, &implantpb.WmiMethodRequest{
		Namespace:  `ROOT\CIMV2`,
		ClassName:  "Win32_Process",
		MethodName: "Create",
		Params: map[string]string{
			"CommandLine": "cmd.exe /c whoami",
		},
	})
	if err != nil {
		t.Fatalf("WmiExecute failed: %v", err)
	}
	wmiExecRequest := waitModuleRequest(t, f.mock, consts.ModuleWmiExec, wmiExecBefore)
	if got := wmiExecRequest.GetSpite().GetWmiMethodRequest().GetMethodName(); got != "Create" {
		t.Fatalf("wmi exec method = %q", got)
	}
	wmiExecContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, wmiExecTask.TaskId)
	if got := wmiExecContent.GetSpite().GetResponse().GetOutput(); !strings.Contains(got, "ReturnValue=0") {
		t.Fatalf("wmi exec response = %q", got)
	}

	runasBefore := len(f.mock.RequestsByName(consts.ModuleRunas))
	runasTask, err := f.rpc.Runas(f.session, &implantpb.RunAsRequest{
		Username: "svc-backup",
		Domain:   "MOCK",
		Program:  "cmd.exe",
		Args:     "/c whoami",
		Netonly:  true,
	})
	if err != nil {
		t.Fatalf("Runas failed: %v", err)
	}
	runasRequest := waitModuleRequest(t, f.mock, consts.ModuleRunas, runasBefore)
	if got := runasRequest.GetSpite().GetRunasRequest().GetUsername(); got != "svc-backup" {
		t.Fatalf("runas username = %q", got)
	}
	runasContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, runasTask.TaskId)
	if got := string(runasContent.GetSpite().GetExecResponse().GetStdout()); !strings.Contains(got, "created process cmd.exe as svc-backup") {
		t.Fatalf("runas stdout = %q", got)
	}
	if got := runasContent.GetSpite().GetExecResponse().GetPid(); got != 5150 {
		t.Fatalf("runas pid = %d, want 5150", got)
	}

	privsBefore := len(f.mock.RequestsByName(consts.ModulePrivs))
	privsTask, err := f.rpc.Privs(f.session, &implantpb.Request{Name: consts.ModulePrivs})
	if err != nil {
		t.Fatalf("Privs failed: %v", err)
	}
	waitModuleRequest(t, f.mock, consts.ModulePrivs, privsBefore)
	privsContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, privsTask.TaskId)
	if got := privsContent.GetSpite().GetResponse().GetArray(); len(got) < 2 || got[0] != "SeDebugPrivilege" {
		t.Fatalf("privs response = %#v", got)
	}

	getSystemBefore := len(f.mock.RequestsByName(consts.ModuleGetSystem))
	getSystemTask, err := f.rpc.GetSystem(f.session, &implantpb.Request{Name: consts.ModuleGetSystem})
	if err != nil {
		t.Fatalf("GetSystem failed: %v", err)
	}
	getSystemRequest := waitModuleRequest(t, f.mock, consts.ModuleGetSystem, getSystemBefore)
	if got := getSystemRequest.GetSpite().GetRequest().GetName(); got != consts.ModuleGetSystem {
		t.Fatalf("getsystem name = %q, want %q", got, consts.ModuleGetSystem)
	}
	getSystemContent := waitTaskFinish(t, f.rpc, f.mock.SessionID, getSystemTask.TaskId)
	if got := getSystemContent.GetSpite().GetResponse().GetOutput(); got != "ok" {
		t.Fatalf("getsystem output = %q, want ok", got)
	}

	killBefore := len(f.mock.RequestsByName(consts.ModuleKill))
	killTask, err := f.rpc.Kill(f.session, &implantpb.Request{
		Name:  consts.ModuleKill,
		Input: "3984",
	})
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	killRequest := waitModuleRequest(t, f.mock, consts.ModuleKill, killBefore)
	if got := killRequest.GetSpite().GetRequest().GetInput(); got != "3984" {
		t.Fatalf("kill input = %q, want 3984", got)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, killTask.TaskId)

	bypassBefore := len(f.mock.RequestsByName(consts.ModuleBypass))
	bypassTask, err := f.rpc.Bypass(f.session, &implantpb.BypassRequest{
		ETW:      true,
		AMSI:     true,
		BlockDll: true,
	})
	if err != nil {
		t.Fatalf("Bypass failed: %v", err)
	}
	bypassRequest := waitModuleRequest(t, f.mock, consts.ModuleBypass, bypassBefore)
	if !bypassRequest.GetSpite().GetBypassRequest().GetETW() || !bypassRequest.GetSpite().GetBypassRequest().GetAMSI() {
		t.Fatalf("bypass request = %#v", bypassRequest.GetSpite().GetBypassRequest())
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, bypassTask.TaskId)

	revBefore := len(f.mock.RequestsByName(consts.ModuleRev2Self))
	revTask, err := f.rpc.Rev2Self(f.session, &implantpb.Request{Name: consts.ModuleRev2Self})
	if err != nil {
		t.Fatalf("Rev2Self failed: %v", err)
	}
	revRequest := waitModuleRequest(t, f.mock, consts.ModuleRev2Self, revBefore)
	if got := revRequest.GetSpite().GetRequest().GetName(); got != consts.ModuleRev2Self {
		t.Fatalf("rev2self name = %q, want %q", got, consts.ModuleRev2Self)
	}
	waitTaskFinish(t, f.rpc, f.mock.SessionID, revTask.TaskId)

	if errs := f.mock.Errors(); len(errs) != 0 {
		t.Fatalf("mock implant async errors = %v", errs)
	}
}
