//go:build mockimplant

package testsupport

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

type MockScenarioLibrary struct {
	mu sync.Mutex

	HomeDir      string
	WorkDir      string
	NotesPath    string
	PayloadPath  string
	RegistryHive string
	RegistryPath string
	ServiceName  string
	ScheduleName string

	sysInfo *implantpb.SysInfo

	dirEntries     map[string]map[string]*implantpb.FileInfo
	fileContents   map[string][]byte
	env            map[string]string
	modules        []string
	addons         []*implantpb.Addon
	services       map[string]*implantpb.Service
	schedules      map[string]*implantpb.TaskSchedule
	registryKeys   map[string][]string
	registryValues map[string]map[string]string
	processes      []*implantpb.Process
	netstat        []*implantpb.SockTabEntry
	drives         []*implantpb.DriveInfo
}

func NewMockScenarioLibrary() *MockScenarioLibrary {
	homeDir := `C:\Users\operator`
	workDir := homeDir + `\workspace`
	notesPath := workDir + `\notes.txt`
	payloadPath := workDir + `\bin\payload.bin`
	registryHive := "HKLM"
	registryPath := `SOFTWARE\Malice`
	serviceName := "MaliceUpdater"
	scheduleName := "MaliceDailyCheckin"

	s := &MockScenarioLibrary{
		HomeDir:        homeDir,
		WorkDir:        workDir,
		NotesPath:      notesPath,
		PayloadPath:    payloadPath,
		RegistryHive:   registryHive,
		RegistryPath:   registryPath,
		ServiceName:    serviceName,
		ScheduleName:   scheduleName,
		dirEntries:     map[string]map[string]*implantpb.FileInfo{},
		fileContents:   map[string][]byte{},
		env:            map[string]string{},
		services:       map[string]*implantpb.Service{},
		schedules:      map[string]*implantpb.TaskSchedule{},
		registryKeys:   map[string][]string{},
		registryValues: map[string]map[string]string{},
	}

	s.sysInfo = &implantpb.SysInfo{
		Filepath:    workDir + `\mock.exe`,
		Workdir:     workDir,
		IsPrivilege: true,
		Os: &implantpb.Os{
			Name:       "windows",
			Version:    "10.0.22631",
			Release:    "23H2",
			Arch:       "amd64",
			Username:   "operator",
			Hostname:   "mock-host",
			Locale:     "Asia/Shanghai",
			ClrVersion: []string{"v4.0.30319"},
		},
		Process: &implantpb.Process{
			Name:  "mock.exe",
			Pid:   4242,
			Ppid:  888,
			Owner: `MOCK\operator`,
			Arch:  "amd64",
			Path:  workDir + `\mock.exe`,
			Args:  "--profile teamserver",
			Uid:   "S-1-5-21-mock",
		},
	}

	s.modules = []string{
		consts.ModuleSysInfo,
		consts.ModulePing,
		consts.ModuleSleep,
		consts.ModuleKeepalive,
		consts.ModuleSwitch,
		consts.ModuleSuicide,
		consts.ModuleClear,
		consts.ModuleKill,
		consts.ModuleBypass,
		consts.ModulePwd,
		consts.ModuleCd,
		consts.ModuleLs,
		consts.ModuleCat,
		consts.ModuleMkdir,
		consts.ModuleTouch,
		consts.ModuleRm,
		consts.ModuleMv,
		consts.ModuleCp,
		consts.ModuleChmod,
		consts.ModuleChown,
		consts.ModuleEnumDrivers,
		consts.ModulePs,
		consts.ModuleEnv,
		consts.ModuleSetEnv,
		consts.ModuleUnsetEnv,
		consts.ModuleWhoami,
		consts.ModuleNetstat,
		consts.ModuleRegQuery,
		consts.ModuleRegListKey,
		consts.ModuleRegListValue,
		consts.ModuleRegAdd,
		consts.ModuleRegDelete,
		consts.ModuleServiceList,
		consts.ModuleServiceQuery,
		consts.ModuleServiceCreate,
		consts.ModuleServiceStart,
		consts.ModuleServiceStop,
		consts.ModuleServiceDelete,
		consts.ModuleTaskSchdList,
		consts.ModuleTaskSchdQuery,
		consts.ModuleTaskSchdCreate,
		consts.ModuleTaskSchdStart,
		consts.ModuleTaskSchdStop,
		consts.ModuleTaskSchdDelete,
		consts.ModuleTaskSchdRun,
		consts.ModuleListModule,
		consts.ModuleRefreshModule,
		consts.ModuleLoadModule,
		consts.ModuleListAddon,
		consts.ModuleLoadAddon,
		consts.ModuleExecuteAddon,
		consts.ModuleRequest,
		consts.ModuleWmiQuery,
		consts.ModuleWmiExec,
		consts.ModuleRunas,
		consts.ModuleRev2Self,
		consts.ModulePrivs,
		consts.ModuleGetSystem,
		consts.ModuleListTask,
		consts.ModuleQueryTask,
		consts.ModuleExecute,
	}
	s.addons = []*implantpb.Addon{
		{Name: "seatbelt", Type: "bof", Depend: consts.ModuleExecute},
		{Name: "sharpview", Type: "assembly", Depend: consts.ModuleExecute},
	}

	s.env["COMPUTERNAME"] = "mock-host"
	s.env["USERNAME"] = "operator"
	s.env["USERDOMAIN"] = "MOCK"
	s.env["PROCESSOR_ARCHITECTURE"] = "AMD64"
	s.env["TEMP"] = homeDir + `\AppData\Local\Temp`
	s.env["MALICE_PROFILE"] = "teamserver"

	s.processes = []*implantpb.Process{
		{
			Name:  "System",
			Pid:   4,
			Ppid:  0,
			Owner: "NT AUTHORITY\\SYSTEM",
			Arch:  "x64",
			Path:  `C:\Windows\System32\ntoskrnl.exe`,
		},
		{
			Name:  "explorer.exe",
			Pid:   3984,
			Ppid:  888,
			Owner: `MOCK\operator`,
			Arch:  "x64",
			Path:  `C:\Windows\explorer.exe`,
		},
		{
			Name:  "mock.exe",
			Pid:   4242,
			Ppid:  3984,
			Owner: `MOCK\operator`,
			Arch:  "x64",
			Path:  workDir + `\mock.exe`,
			Args:  "--profile teamserver",
		},
	}

	s.netstat = []*implantpb.SockTabEntry{
		{
			LocalAddr:  "127.0.0.1:8443",
			RemoteAddr: "0.0.0.0:0",
			SkState:    "LISTEN",
			Pid:        "4242",
			Protocol:   "tcp",
		},
		{
			LocalAddr:  "192.168.56.11:49822",
			RemoteAddr: "104.26.10.78:443",
			SkState:    "ESTABLISHED",
			Pid:        "4242",
			Protocol:   "tcp",
		},
	}

	s.drives = []*implantpb.DriveInfo{
		{
			Path:       `C:\`,
			DriveType:  "Fixed drive",
			TotalSize:  512 * 1024 * 1024 * 1024,
			FreeSize:   213 * 1024 * 1024 * 1024,
			FileSystem: "NTFS",
		},
		{
			Path:       `Z:\`,
			DriveType:  "Network drive",
			TotalSize:  1024 * 1024 * 1024,
			FreeSize:   512 * 1024 * 1024,
			FileSystem: "SMB",
		},
	}

	s.services[s.normName(serviceName)] = &implantpb.Service{
		Config: &implantpb.ServiceConfig{
			Name:           serviceName,
			DisplayName:    "Malice Updater",
			ExecutablePath: `C:\Program Files\Malice\updater.exe`,
			StartType:      2,
			ErrorControl:   1,
			AccountName:    `LocalSystem`,
		},
		Status: &implantpb.ServiceStatus{
			CurrentState: 4,
			ProcessId:    4242,
			ExitCode:     0,
		},
	}

	s.schedules[s.normName(scheduleName)] = &implantpb.TaskSchedule{
		Name:           scheduleName,
		Path:           `\Malice`,
		ExecutablePath: `C:\Program Files\Malice\beacon.exe`,
		TriggerType:    2,
		StartBoundary:  "2026-03-14T09:00:00",
		Description:    "Daily check-in task for mock implant coverage",
		Enabled:        true,
		LastRunTime:    "2026-03-14T08:58:00",
		NextRunTime:    "2026-03-15T09:00:00",
	}

	rootRegKey := s.registryKey(registryHive, `SOFTWARE`)
	appRegKey := s.registryKey(registryHive, registryPath)
	s.registryKeys[rootRegKey] = []string{"Malice", "Microsoft"}
	s.registryKeys[appRegKey] = []string{"Modules", "Runtime"}
	s.registryValues[appRegKey] = map[string]string{
		"InstallPath": `C:\Program Files\Malice`,
		"Version":     "1.4.2",
		"Channel":     "stable",
	}

	s.ensureDir(`C:\`)
	s.ensureDir(homeDir)
	s.ensureDir(workDir)
	s.ensureDir(workDir + `\bin`)
	s.ensureDir(homeDir + `\Downloads`)
	s.setFile(notesPath, []byte("operator notes: validate rpc coverage through mock implant\n"))
	s.setFile(payloadPath, []byte{0x4d, 0x5a, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00})
	s.setFile(homeDir+`\Downloads\report.log`, []byte("report ready\n"))

	return s
}

func (s *MockScenarioLibrary) Install(mock *MockImplant) {
	if mock == nil {
		return
	}
	if mock.Register != nil {
		mock.Register.Sysinfo = proto.Clone(s.sysInfo).(*implantpb.SysInfo)
		mock.Register.Module = append([]string(nil), s.modules...)
		mock.Register.Addons = s.cloneAddonsLocked()
	}

	mock.On(consts.ModuleSysInfo, s.handleInfo)
	mock.On(consts.ModulePing, s.handlePing)
	mock.On(consts.ModuleSleep, s.handleEmpty)
	mock.On(consts.ModuleKeepalive, s.handleKeepalive)
	mock.On(consts.ModuleSwitch, s.handleEmpty)
	mock.On(consts.ModuleSuicide, s.handleEmpty)
	mock.On(consts.ModuleClear, s.handleEmpty)
	mock.On(consts.ModuleKill, s.handleEmpty)
	mock.On(consts.ModuleBypass, s.handleEmpty)

	mock.On(consts.ModulePwd, s.handlePwd)
	mock.On(consts.ModuleCd, s.handleCd)
	mock.On(consts.ModuleLs, s.handleLs)
	mock.On(consts.ModuleCat, s.handleCat)
	mock.On(consts.ModuleMkdir, s.handleMkdir)
	mock.On(consts.ModuleTouch, s.handleTouch)
	mock.On(consts.ModuleRm, s.handleRm)
	mock.On(consts.ModuleMv, s.handleMv)
	mock.On(consts.ModuleCp, s.handleCp)
	mock.On(consts.ModuleChmod, s.handleEmpty)
	mock.On(consts.ModuleChown, s.handleResponseStatus)
	mock.On(consts.ModuleEnumDrivers, s.handleEnumDrivers)

	mock.On(consts.ModulePs, s.handlePs)
	mock.On(consts.ModuleEnv, s.handleEnv)
	mock.On(consts.ModuleSetEnv, s.handleSetEnv)
	mock.On(consts.ModuleUnsetEnv, s.handleUnsetEnv)
	mock.On(consts.ModuleWhoami, s.handleWhoami)
	mock.On(consts.ModuleNetstat, s.handleNetstat)

	mock.On(consts.ModuleRegQuery, s.handleRegQuery)
	mock.On(consts.ModuleRegListKey, s.handleRegListKey)
	mock.On(consts.ModuleRegListValue, s.handleRegListValue)
	mock.On(consts.ModuleRegAdd, s.handleRegAdd)
	mock.On(consts.ModuleRegDelete, s.handleRegDelete)

	mock.On(consts.ModuleServiceList, s.handleServiceList)
	mock.On(consts.ModuleServiceQuery, s.handleServiceQuery)
	mock.On(consts.ModuleServiceCreate, s.handleServiceCreate)
	mock.On(consts.ModuleServiceStart, s.handleServiceStart)
	mock.On(consts.ModuleServiceStop, s.handleServiceStop)
	mock.On(consts.ModuleServiceDelete, s.handleServiceDelete)

	mock.On(consts.ModuleTaskSchdList, s.handleTaskSchdList)
	mock.On(consts.ModuleTaskSchdQuery, s.handleTaskSchdQuery)
	mock.On(consts.ModuleTaskSchdCreate, s.handleTaskSchdCreate)
	mock.On(consts.ModuleTaskSchdStart, s.handleTaskSchdStart)
	mock.On(consts.ModuleTaskSchdStop, s.handleTaskSchdStop)
	mock.On(consts.ModuleTaskSchdDelete, s.handleTaskSchdDelete)
	mock.On(consts.ModuleTaskSchdRun, s.handleTaskSchdRun)

	mock.On(consts.ModuleListModule, s.handleListModule)
	mock.On(consts.ModuleRefreshModule, s.handleListModule)
	mock.On(consts.ModuleLoadModule, s.handleLoadModule)
	mock.On(consts.ModuleListAddon, s.handleListAddon)
	mock.On(consts.ModuleLoadAddon, s.handleLoadAddon)
	mock.On(consts.ModuleExecuteAddon, s.handleExecuteAddon)

	mock.On(consts.ModuleRequest, s.handleCurl)
	mock.On(consts.ModuleWmiQuery, s.handleWmiQuery)
	mock.On(consts.ModuleWmiExec, s.handleWmiExecute)
	mock.On(consts.ModuleRunas, s.handleRunAs)
	mock.On(consts.ModuleRev2Self, s.handleEmpty)
	mock.On(consts.ModulePrivs, s.handlePrivileges)
	mock.On(consts.ModuleGetSystem, s.handleResponseStatus)
	mock.On(consts.ModuleListTask, s.handleListTasks)
	mock.On(consts.ModuleQueryTask, s.handleQueryTask)
	mock.On(consts.ModuleExecute, s.handleExecute)
}

func (s *MockScenarioLibrary) handleInfo(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Sysinfo{Sysinfo: proto.Clone(s.sysInfo).(*implantpb.SysInfo)},
	})
}

func (s *MockScenarioLibrary) handlePing(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	ping := &implantpb.Ping{}
	if req.GetSpite() != nil && req.GetSpite().GetPing() != nil {
		ping = proto.Clone(req.GetSpite().GetPing()).(*implantpb.Ping)
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Ping{Ping: ping},
	})
}

func (s *MockScenarioLibrary) handleKeepalive(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	resp := &implantpb.CommonBody{Name: consts.ModuleKeepalive}
	if req.GetSpite() != nil && req.GetSpite().GetCommon() != nil {
		resp.BoolArray = append(resp.BoolArray, req.GetSpite().GetCommon().GetBoolArray()...)
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Common{Common: resp},
	})
}

func (s *MockScenarioLibrary) handleEmpty(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
	})
}

func (s *MockScenarioLibrary) handleResponseStatus(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{Output: "ok"}},
	})
}

func (s *MockScenarioLibrary) handlePwd(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{Output: s.WorkDir}},
	})
}

func (s *MockScenarioLibrary) handleCd(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.WorkDir
	if command := req.GetSpite().GetRequest(); command != nil && command.GetInput() != "" {
		target = s.normPath(command.GetInput())
	}
	s.ensureDir(target)
	s.WorkDir = target
	s.sysInfo.Workdir = target
	s.sysInfo.Filepath = target + `\mock.exe`
	if s.sysInfo.Process != nil {
		s.sysInfo.Process.Path = s.sysInfo.Filepath
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{Output: target}},
	})
}

func (s *MockScenarioLibrary) handleLs(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.WorkDir
	if command := req.GetSpite().GetRequest(); command != nil && command.GetInput() != "" {
		target = s.normPath(command.GetInput())
	}
	response := &implantpb.LsResponse{
		Path:   target,
		Exists: true,
		Files:  s.cloneDirEntriesLocked(target),
	}
	if _, ok := s.dirEntries[s.normPath(target)]; !ok {
		response.Exists = false
		response.Files = nil
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_LsResponse{LsResponse: response},
	})
}

func (s *MockScenarioLibrary) handleCat(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.NotesPath
	if command := req.GetSpite().GetRequest(); command != nil && command.GetInput() != "" {
		path = s.normPath(command.GetInput())
	}
	content := append([]byte(nil), s.fileContents[s.normPath(path)]...)
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_BinaryResponse{BinaryResponse: &implantpb.BinaryResponse{Data: content, Status: 200}},
	})
}

func (s *MockScenarioLibrary) handleMkdir(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if command := req.GetSpite().GetRequest(); command != nil && command.GetInput() != "" {
		s.ensureDir(command.GetInput())
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleTouch(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if command := req.GetSpite().GetRequest(); command != nil && command.GetInput() != "" {
		s.setFile(command.GetInput(), nil)
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleRm(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if command := req.GetSpite().GetRequest(); command != nil && command.GetInput() != "" {
		s.removePath(command.GetInput())
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleMv(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if command := req.GetSpite().GetRequest(); command != nil && len(command.GetArgs()) >= 2 {
		s.movePath(command.GetArgs()[0], command.GetArgs()[1])
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleCp(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if command := req.GetSpite().GetRequest(); command != nil && len(command.GetArgs()) >= 2 {
		s.copyPath(command.GetArgs()[0], command.GetArgs()[1])
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleEnumDrivers(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_EnumDriversResponse{EnumDriversResponse: &implantpb.EnumDriversResponse{
			Drives: s.cloneDrivesLocked(),
		}},
	})
}

func (s *MockScenarioLibrary) handlePs(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_PsResponse{PsResponse: &implantpb.PsResponse{
			Processes: s.cloneProcessesLocked(),
		}},
	})
}

func (s *MockScenarioLibrary) handleEnv(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{
			Kv: s.cloneStringMapLocked(s.env),
		}},
	})
}

func (s *MockScenarioLibrary) handleSetEnv(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if command := req.GetSpite().GetRequest(); command != nil && len(command.GetArgs()) >= 2 {
		s.env[command.GetArgs()[0]] = command.GetArgs()[1]
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleUnsetEnv(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if command := req.GetSpite().GetRequest(); command != nil {
		name := command.GetInput()
		if name == "" && len(command.GetArgs()) > 0 {
			name = command.GetArgs()[0]
		}
		delete(s.env, name)
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleWhoami(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{
			Output: fmt.Sprintf("%s\\%s", s.sysInfo.GetOs().GetHostname(), s.sysInfo.GetOs().GetUsername()),
		}},
	})
}

func (s *MockScenarioLibrary) handleNetstat(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_NetstatResponse{NetstatResponse: &implantpb.NetstatResponse{
			Socks: s.cloneSocksLocked(),
		}},
	})
}

func (s *MockScenarioLibrary) handleRegQuery(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetRegistryRequest()
	if request == nil {
		return s.handleResponseStatus(ctx, req, send)
	}
	key := s.registryKey(request.GetHive(), request.GetPath())
	value := ""
	if values, ok := s.registryValues[key]; ok {
		value = values[request.GetKey()]
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{
			Output: value,
			Kv: map[string]string{
				request.GetKey(): value,
			},
		}},
	})
}

func (s *MockScenarioLibrary) handleRegListKey(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetRegistryRequest()
	array := []string{}
	if request != nil {
		key := s.registryKey(request.GetHive(), request.GetPath())
		array = append(array, s.registryKeys[key]...)
	}
	sort.Strings(array)
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{
			Array: array,
		}},
	})
}

func (s *MockScenarioLibrary) handleRegListValue(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetRegistryRequest()
	values := map[string]string{}
	if request != nil {
		key := s.registryKey(request.GetHive(), request.GetPath())
		if stored, ok := s.registryValues[key]; ok {
			values = s.cloneStringMapLocked(stored)
		}
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{
			Kv: values,
		}},
	})
}

func (s *MockScenarioLibrary) handleRegAdd(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetRegistryWriteRequest()
	if request != nil {
		key := s.registryKey(request.GetHive(), request.GetPath())
		if _, ok := s.registryValues[key]; !ok {
			s.registryValues[key] = map[string]string{}
		}
		switch {
		case request.GetStringValue() != "":
			s.registryValues[key][request.GetKey()] = request.GetStringValue()
		case len(request.GetByteValue()) > 0:
			s.registryValues[key][request.GetKey()] = fmt.Sprintf("%x", request.GetByteValue())
		case request.GetQwordValue() != 0:
			s.registryValues[key][request.GetKey()] = fmt.Sprintf("%d", request.GetQwordValue())
		default:
			s.registryValues[key][request.GetKey()] = fmt.Sprintf("%d", request.GetDwordValue())
		}
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleRegDelete(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetRegistryRequest()
	if request != nil {
		key := s.registryKey(request.GetHive(), request.GetPath())
		if request.GetKey() == "" {
			delete(s.registryValues, key)
			delete(s.registryKeys, key)
		} else if values, ok := s.registryValues[key]; ok {
			delete(values, request.GetKey())
		}
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleServiceList(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_ServicesResponse{ServicesResponse: &implantpb.ServicesResponse{
			Services: s.cloneServicesLocked(),
		}},
	})
}

func (s *MockScenarioLibrary) handleServiceQuery(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetServiceRequest()
	service := s.services[s.normName(s.ServiceName)]
	if request != nil && request.GetName() != "" {
		if candidate, ok := s.services[s.normName(request.GetName())]; ok {
			service = candidate
		}
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_ServiceResponse{ServiceResponse: proto.Clone(service).(*implantpb.Service)},
	})
}

func (s *MockScenarioLibrary) handleServiceCreate(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetServiceRequest()
	if request != nil {
		service := &implantpb.Service{
			Config: proto.Clone(request).(*implantpb.ServiceConfig),
			Status: &implantpb.ServiceStatus{
				CurrentState: 1,
				ProcessId:    0,
				ExitCode:     0,
			},
		}
		s.services[s.normName(request.GetName())] = service
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleServiceStart(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateServiceStateLocked(req.GetSpite().GetServiceRequest(), 4, 4242)
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleServiceStop(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateServiceStateLocked(req.GetSpite().GetServiceRequest(), 1, 0)
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleServiceDelete(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetServiceRequest()
	if request != nil {
		delete(s.services, s.normName(request.GetName()))
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleTaskSchdList(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_SchedulesResponse{SchedulesResponse: &implantpb.TaskSchedulesResponse{
			Schedules: s.cloneSchedulesLocked(),
		}},
	})
}

func (s *MockScenarioLibrary) handleTaskSchdQuery(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetScheduleRequest()
	schedule := s.schedules[s.normName(s.ScheduleName)]
	if request != nil && request.GetName() != "" {
		if candidate, ok := s.schedules[s.normName(request.GetName())]; ok {
			schedule = candidate
		}
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_ScheduleResponse{ScheduleResponse: proto.Clone(schedule).(*implantpb.TaskSchedule)},
	})
}

func (s *MockScenarioLibrary) handleTaskSchdCreate(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetScheduleRequest()
	if request != nil {
		schedule := proto.Clone(request).(*implantpb.TaskSchedule)
		if schedule.Description == "" {
			schedule.Description = "Created by mock implant scenario library"
		}
		schedule.Enabled = true
		if schedule.LastRunTime == "" {
			schedule.LastRunTime = "never"
		}
		if schedule.NextRunTime == "" {
			schedule.NextRunTime = schedule.StartBoundary
		}
		s.schedules[s.normName(schedule.GetName())] = schedule
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleTaskSchdStart(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateScheduleEnabledLocked(req.GetSpite().GetScheduleRequest(), true)
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleTaskSchdStop(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateScheduleEnabledLocked(req.GetSpite().GetScheduleRequest(), false)
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleTaskSchdDelete(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetScheduleRequest()
	if request != nil {
		delete(s.schedules, s.normName(request.GetName()))
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleTaskSchdRun(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetScheduleRequest()
	if request != nil {
		if schedule, ok := s.schedules[s.normName(request.GetName())]; ok {
			schedule.LastRunTime = "2026-03-14T12:34:56"
		}
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleListModule(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Modules{Modules: &implantpb.Modules{
			Modules: append([]string(nil), s.modules...),
		}},
	})
}

func (s *MockScenarioLibrary) handleLoadModule(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetLoadModule()
	moduleName := "custom.module"
	if request != nil && request.GetBundle() != "" {
		moduleName = request.GetBundle()
	}
	if !containsString(s.modules, moduleName) {
		s.modules = append(s.modules, moduleName)
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Modules{Modules: &implantpb.Modules{
			Modules: []string{moduleName},
		}},
	})
}

func (s *MockScenarioLibrary) handleListAddon(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Addons{Addons: &implantpb.Addons{
			Addons: s.cloneAddonsLocked(),
		}},
	})
}

func (s *MockScenarioLibrary) handleLoadAddon(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := req.GetSpite().GetLoadAddon()
	if request != nil {
		s.addons = append(s.addons, &implantpb.Addon{
			Name:   request.GetName(),
			Type:   request.GetType(),
			Depend: request.GetDepend(),
		})
	}
	return s.handleEmpty(ctx, req, send)
}

func (s *MockScenarioLibrary) handleExecuteAddon(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	request := req.GetSpite().GetExecuteAddon()
	name := "addon"
	if request != nil && request.GetAddon() != "" {
		name = request.GetAddon()
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_BinaryResponse{BinaryResponse: &implantpb.BinaryResponse{
			Data:   []byte("addon:" + name + ":ok"),
			Status: 200,
		}},
	})
}

func (s *MockScenarioLibrary) handleCurl(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	method := "GET"
	url := "https://example.invalid/"
	if request := req.GetSpite().GetCurlRequest(); request != nil {
		if request.GetMethod() != "" {
			method = request.GetMethod()
		}
		if request.GetUrl() != "" {
			url = request.GetUrl()
		}
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_BinaryResponse{BinaryResponse: &implantpb.BinaryResponse{
			Data:   []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nmock-curl %s %s", method, url)),
			Status: 200,
		}},
	})
}

func (s *MockScenarioLibrary) handleWmiQuery(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	namespace := "ROOT\\CIMV2"
	query := "SELECT Caption,Version FROM Win32_OperatingSystem"
	if request := req.GetSpite().GetWmiRequest(); request != nil {
		if request.GetNamespace() != "" {
			namespace = request.GetNamespace()
		}
		if len(request.GetArgs()) > 0 {
			query = strings.Join(request.GetArgs(), " ")
		}
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{
			Output: fmt.Sprintf("Namespace=%s;Query=%s;Caption=Microsoft Windows 11 Enterprise;Version=10.0.22631", namespace, query),
		}},
	})
}

func (s *MockScenarioLibrary) handleWmiExecute(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	className := "Win32_Process"
	methodName := "Create"
	if request := req.GetSpite().GetWmiMethodRequest(); request != nil {
		if request.GetClassName() != "" {
			className = request.GetClassName()
		}
		if request.GetMethodName() != "" {
			methodName = request.GetMethodName()
		}
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{
			Output: fmt.Sprintf("Class=%s;Method=%s;ReturnValue=0", className, methodName),
		}},
	})
}

func (s *MockScenarioLibrary) handleRunAs(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	request := req.GetSpite().GetRunasRequest()
	username := "operator"
	program := "cmd.exe"
	if request != nil && request.GetUsername() != "" {
		username = request.GetUsername()
	}
	if request != nil && request.GetProgram() != "" {
		program = request.GetProgram()
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
			StatusCode: 0,
			Stdout:     []byte(fmt.Sprintf("created process %s as %s", program, username)),
			Pid:        5150,
			End:        true,
		}},
	})
}

func (s *MockScenarioLibrary) handlePrivileges(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_Response{Response: &implantpb.Response{
			Array: []string{"SeDebugPrivilege", "SeImpersonatePrivilege", "SeIncreaseQuotaPrivilege"},
		}},
	})
}

func (s *MockScenarioLibrary) handleListTasks(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_TaskList{TaskList: &implantpb.TaskListResponse{
			Tasks: []*implantpb.TaskInfo{
				{TaskId: 11, Last: 1710396000, RecvCount: 1, SendCount: 1},
				{TaskId: 12, Last: 1710399600, RecvCount: 2, SendCount: 2},
			},
		}},
	})
}

func (s *MockScenarioLibrary) handleQueryTask(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	taskID := uint32(11)
	if request := req.GetSpite().GetTask(); request != nil && request.GetTaskId() != 0 {
		taskID = request.GetTaskId()
	}
	return send(&implantpb.Spite{
		Body: &implantpb.Spite_TaskInfo{TaskInfo: &implantpb.TaskInfo{
			TaskId:    taskID,
			Last:      1710399600,
			RecvCount: 2,
			SendCount: 2,
		}},
	})
}

func (s *MockScenarioLibrary) handleExecute(ctx context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
	request := req.GetSpite().GetExecRequest()
	realtime := request != nil && request.GetRealtime()
	output := request != nil && request.GetOutput()

	s.mu.Lock()
	chunks, stdout := s.execOutputLocked(request)
	s.mu.Unlock()

	if realtime && output {
		return SendRealisticExecStream(ctx, send, 4242, 0, chunks...)
	}

	return send(&implantpb.Spite{
		Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
			StatusCode: 0,
			Stdout:     stdout,
			Pid:        4242,
			End:        true,
		}},
	})
}

func (s *MockScenarioLibrary) updateServiceStateLocked(request *implantpb.ServiceConfig, state uint32, pid uint32) {
	if request == nil {
		return
	}
	service, ok := s.services[s.normName(request.GetName())]
	if !ok {
		return
	}
	service.Status.CurrentState = state
	service.Status.ProcessId = pid
}

func (s *MockScenarioLibrary) updateScheduleEnabledLocked(request *implantpb.TaskSchedule, enabled bool) {
	if request == nil {
		return
	}
	schedule, ok := s.schedules[s.normName(request.GetName())]
	if !ok {
		return
	}
	schedule.Enabled = enabled
}

func (s *MockScenarioLibrary) ensureDir(path string) {
	path = s.normPath(path)
	if path == "" {
		return
	}
	if _, ok := s.dirEntries[path]; ok {
		return
	}
	s.dirEntries[path] = map[string]*implantpb.FileInfo{}
	parent := s.normPath(filepath.Dir(path))
	if parent != "" && parent != path {
		s.ensureDir(parent)
		name := filepath.Base(path)
		s.dirEntries[parent][name] = &implantpb.FileInfo{
			Name:    name,
			IsDir:   true,
			Size:    0,
			ModTime: time.Now().Add(-2 * time.Hour).Unix(),
			Mode:    0o755,
		}
	}
}

func (s *MockScenarioLibrary) setFile(path string, data []byte) {
	path = s.normPath(path)
	parent := s.normPath(filepath.Dir(path))
	s.ensureDir(parent)
	name := filepath.Base(path)
	s.dirEntries[parent][name] = &implantpb.FileInfo{
		Name:    name,
		IsDir:   false,
		Size:    uint64(len(data)),
		ModTime: time.Now().Add(-15 * time.Minute).Unix(),
		Mode:    0o644,
	}
	s.fileContents[path] = append([]byte(nil), data...)
}

func (s *MockScenarioLibrary) removePath(path string) {
	path = s.normPath(path)
	parent := s.normPath(filepath.Dir(path))
	name := filepath.Base(path)
	delete(s.fileContents, path)
	delete(s.dirEntries, path)
	if entries, ok := s.dirEntries[parent]; ok {
		delete(entries, name)
	}
}

func (s *MockScenarioLibrary) copyPath(src, dst string) {
	src = s.normPath(src)
	dst = s.normPath(dst)
	if content, ok := s.fileContents[src]; ok {
		s.setFile(dst, content)
	}
}

func (s *MockScenarioLibrary) movePath(src, dst string) {
	src = s.normPath(src)
	dst = s.normPath(dst)
	if content, ok := s.fileContents[src]; ok {
		s.setFile(dst, content)
		s.removePath(src)
	}
}

func (s *MockScenarioLibrary) cloneDirEntriesLocked(path string) []*implantpb.FileInfo {
	entries, ok := s.dirEntries[s.normPath(path)]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*implantpb.FileInfo, 0, len(entries))
	for _, name := range names {
		out = append(out, proto.Clone(entries[name]).(*implantpb.FileInfo))
	}
	return out
}

func (s *MockScenarioLibrary) cloneProcessesLocked() []*implantpb.Process {
	out := make([]*implantpb.Process, 0, len(s.processes))
	for _, process := range s.processes {
		out = append(out, proto.Clone(process).(*implantpb.Process))
	}
	return out
}

func (s *MockScenarioLibrary) cloneSocksLocked() []*implantpb.SockTabEntry {
	out := make([]*implantpb.SockTabEntry, 0, len(s.netstat))
	for _, sock := range s.netstat {
		out = append(out, proto.Clone(sock).(*implantpb.SockTabEntry))
	}
	return out
}

func (s *MockScenarioLibrary) cloneDrivesLocked() []*implantpb.DriveInfo {
	out := make([]*implantpb.DriveInfo, 0, len(s.drives))
	for _, drive := range s.drives {
		out = append(out, proto.Clone(drive).(*implantpb.DriveInfo))
	}
	return out
}

func (s *MockScenarioLibrary) cloneServicesLocked() []*implantpb.Service {
	names := make([]string, 0, len(s.services))
	for name := range s.services {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*implantpb.Service, 0, len(s.services))
	for _, name := range names {
		out = append(out, proto.Clone(s.services[name]).(*implantpb.Service))
	}
	return out
}

func (s *MockScenarioLibrary) cloneSchedulesLocked() []*implantpb.TaskSchedule {
	names := make([]string, 0, len(s.schedules))
	for name := range s.schedules {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*implantpb.TaskSchedule, 0, len(s.schedules))
	for _, name := range names {
		out = append(out, proto.Clone(s.schedules[name]).(*implantpb.TaskSchedule))
	}
	return out
}

func (s *MockScenarioLibrary) cloneAddonsLocked() []*implantpb.Addon {
	out := make([]*implantpb.Addon, 0, len(s.addons))
	for _, addon := range s.addons {
		out = append(out, proto.Clone(addon).(*implantpb.Addon))
	}
	return out
}

func (s *MockScenarioLibrary) cloneStringMapLocked(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (s *MockScenarioLibrary) registryKey(hive, path string) string {
	return strings.ToUpper(strings.TrimSpace(hive)) + "|" + s.normName(path)
}

func (s *MockScenarioLibrary) normName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *MockScenarioLibrary) normPath(path string) string {
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

func (s *MockScenarioLibrary) execOutputLocked(request *implantpb.ExecRequest) ([]MockExecChunk, []byte) {
	if request == nil {
		return nil, nil
	}

	stdout := s.renderExecStdoutLocked(request)
	if request.GetRealtime() && request.GetOutput() {
		commandLine := strings.ToLower(strings.Join(request.GetArgs(), " "))
		chunks := make([]MockExecChunk, 0, 2)
		switch {
		case strings.Contains(commandLine, "echo alpha") && strings.Contains(commandLine, "echo omega"):
			chunks = append(chunks,
				MockExecChunk{Delay: 50 * time.Millisecond, Stdout: []byte("alpha\r\n")},
				MockExecChunk{Delay: 50 * time.Millisecond, Stdout: []byte("omega\r\n")},
			)
		case len(stdout) > 0:
			chunks = append(chunks, MockExecChunk{Delay: 50 * time.Millisecond, Stdout: stdout})
		}
		return chunks, nil
	}

	if !request.GetOutput() {
		return nil, nil
	}
	return nil, stdout
}

func (s *MockScenarioLibrary) renderExecStdoutLocked(request *implantpb.ExecRequest) []byte {
	if request == nil {
		return nil
	}

	path := strings.ToLower(strings.TrimSpace(request.GetPath()))
	args := request.GetArgs()
	joinedArgs := strings.Join(args, " ")
	lowerArgs := strings.ToLower(joinedArgs)

	switch {
	case strings.HasSuffix(path, "cmd.exe") && strings.Contains(lowerArgs, "hostname"):
		host := "mock-host"
		if s.sysInfo != nil && s.sysInfo.GetOs() != nil && s.sysInfo.GetOs().GetHostname() != "" {
			host = s.sysInfo.GetOs().GetHostname()
		}
		return []byte(host + "\r\n")
	case strings.HasSuffix(path, "cmd.exe") && strings.Contains(lowerArgs, "echo alpha") && strings.Contains(lowerArgs, "echo omega"):
		return []byte("alpha\r\nomega\r\n")
	case strings.HasSuffix(path, "cmd.exe") && strings.Contains(lowerArgs, "echo "):
		echoIndex := strings.Index(lowerArgs, "echo ")
		if echoIndex >= 0 {
			raw := strings.TrimSpace(joinedArgs[echoIndex+len("echo "):])
			if raw != "" {
				return []byte(raw + "\r\n")
			}
		}
	}

	commandLine := strings.TrimSpace(strings.Join(append([]string{request.GetPath()}, request.GetArgs()...), " "))
	if commandLine == "" {
		return nil
	}
	return []byte(fmt.Sprintf("executed: %s\r\n", commandLine))
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
