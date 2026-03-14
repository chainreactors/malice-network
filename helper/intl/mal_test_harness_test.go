package intl

import (
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

// CapturedCommand holds metadata captured when a Lua script calls command().
type CapturedCommand struct {
	Name       string
	Short      string
	TTP        string
	CobraCmd   *cobra.Command
	LuaFunc    *lua.LFunction
	OpsecScore float64
	HasOpsec   bool
	HelpText   string
}

// CapturedBofPack holds a captured pack_bof_args call.
type CapturedBofPack struct {
	Format string
	Args   []string
}

// CapturedBofCall holds a captured bof() call.
type CapturedBofCall struct {
	Resource string
	Output   bool
}

// MockOs represents mock OS information for a session.
type MockOs struct {
	Arch       string
	Name       string
	Version    string
	ClrVersion []string
}

// MockSession represents a mock implant session.
type MockSession struct {
	Os        *MockOs
	IsAdmin   bool
	SessionId string
}

// TestHarness captures all metadata produced by running Lua plugin scripts
// in a mock VM environment. It provides assertions against command registration,
// OPSEC scores, bof_pack format strings, and resource paths.
type TestHarness struct {
	mu            sync.Mutex
	Commands      map[string]*CapturedCommand
	BofPacks      []CapturedBofPack
	ResourcePaths []string
	BofCalls      []CapturedBofCall
}

// NewTestHarness creates a new TestHarness with empty state.
func NewTestHarness() *TestHarness {
	return &TestHarness{
		Commands: make(map[string]*CapturedCommand),
	}
}

func requireCommunityFixture(t *testing.T, path string) {
	t.Helper()
	if FileExists(path) {
		return
	}
	t.Skipf("community fixture %q not present in repository checkout", path)
}

func readCommunityFixture(path string) ([]byte, error) {
	candidates := []string{
		path,
		"community/" + path,
		"community/community/" + path,
	}
	for _, candidate := range candidates {
		content, err := UnifiedFS.ReadFile(candidate)
		if err == nil {
			return content, nil
		}
	}
	return nil, fmt.Errorf("embedded fixture not found: %s", path)
}

// NewMockVM creates a gopher-lua VM with all standard libraries from mals
// and all Go functions replaced by mock implementations that capture metadata.
func (h *TestHarness) NewMockVM() *lua.LState {
	vm := mals.NewLuaVM()
	h.registerMockFunctions(vm)
	h.addEmbedLoader(vm)
	h.preloadMockPackages(vm)
	return vm
}

// LoadCommunityMain loads and executes community/community/main.lua,
// which triggers require() of all sub-modules and registers all commands.
func (h *TestHarness) LoadCommunityMain(vm *lua.LState) error {
	content, err := readCommunityFixture("main.lua")
	if err != nil {
		return fmt.Errorf("failed to read main.lua: %w", err)
	}
	fn, err := vm.LoadString(string(content))
	if err != nil {
		return fmt.Errorf("failed to compile main.lua: %w", err)
	}
	vm.Push(fn)
	return vm.PCall(0, lua.MultRet, nil)
}

// CallCommandHandler invokes the Lua handler function of a registered command.
// The args are pushed onto the Lua stack as arguments to the handler.
func (h *TestHarness) CallCommandHandler(vm *lua.LState, cmdName string, args ...lua.LValue) error {
	h.mu.Lock()
	cmd, ok := h.Commands[cmdName]
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("command %q not found", cmdName)
	}
	vm.Push(cmd.LuaFunc)
	for _, arg := range args {
		vm.Push(arg)
	}
	return vm.PCall(len(args), lua.MultRet, nil)
}

// GetAllLuaFiles returns all .lua file paths within UnifiedFS.
func GetAllLuaFiles() ([]string, error) {
	var files []string
	err := fs.WalkDir(UnifiedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".lua") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ValidBofPackFormat checks whether every character in format is a valid
// BOF pack format specifier (b=binary, i=int32, s=short, z=ansi, Z=wide, c=char).
func ValidBofPackFormat(format string) bool {
	for _, c := range format {
		switch c {
		case 'b', 'i', 's', 'z', 'Z', 'c':
			// valid
		default:
			return false
		}
	}
	return len(format) > 0
}

// registerMockFunctions registers mock versions of all Go functions that
// Lua plugin scripts call. The mocks capture metadata instead of performing
// real operations.
func (h *TestHarness) registerMockFunctions(vm *lua.LState) {
	// --- Core registration functions ---

	// command(name, fn, short, ttp) -> cobra.Command (wrapped via luar)
	vm.SetGlobal("command", vm.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		fn := L.CheckFunction(2)
		short := L.OptString(3, "")
		ttp := L.OptString(4, "")

		cmd := &cobra.Command{
			Use:   name,
			Short: short,
			Annotations: map[string]string{
				"ttp": ttp,
			},
		}

		h.mu.Lock()
		h.Commands[name] = &CapturedCommand{
			Name:     name,
			Short:    short,
			TTP:      ttp,
			CobraCmd: cmd,
			LuaFunc:  fn,
		}
		h.mu.Unlock()

		L.Push(luar.New(L, cmd))
		return 1
	}))

	// opsec(name, score) -> bool
	vm.SetGlobal("opsec", vm.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		score := float64(L.CheckNumber(2))

		h.mu.Lock()
		if cmd, ok := h.Commands[name]; ok {
			cmd.OpsecScore = score
			cmd.HasOpsec = true
		}
		h.mu.Unlock()

		L.Push(lua.LTrue)
		return 1
	}))

	// help(name, long) -> bool
	vm.SetGlobal("help", vm.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		long := L.CheckString(2)

		h.mu.Lock()
		if cmd, ok := h.Commands[name]; ok {
			cmd.HelpText = long
			cmd.CobraCmd.Long = long
		}
		h.mu.Unlock()

		L.Push(lua.LTrue)
		return 1
	}))

	// example(name, text) -> bool
	vm.SetGlobal("example", vm.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		text := L.CheckString(2)

		h.mu.Lock()
		if cmd, ok := h.Commands[name]; ok {
			cmd.CobraCmd.Example = text
		}
		h.mu.Unlock()

		L.Push(lua.LTrue)
		return 1
	}))

	// --- Resource functions ---

	// script_resource(filename) -> string
	vm.SetGlobal("script_resource", vm.NewFunction(func(L *lua.LState) int {
		filename := L.CheckString(1)
		h.mu.Lock()
		h.ResourcePaths = append(h.ResourcePaths, filename)
		h.mu.Unlock()
		L.Push(lua.LString("mock://" + filename))
		return 1
	}))

	// global_resource(filename) -> string
	vm.SetGlobal("global_resource", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("mock://global/" + L.CheckString(1)))
		return 1
	}))

	// find_resource(session, base, ext) -> string
	vm.SetGlobal("find_resource", vm.NewFunction(func(L *lua.LState) int {
		base := L.CheckString(2)
		ext := L.CheckString(3)
		L.Push(lua.LString(fmt.Sprintf("mock://%s.x64.%s", base, ext)))
		return 1
	}))

	// find_global_resource(session, base, ext) -> string
	vm.SetGlobal("find_global_resource", vm.NewFunction(func(L *lua.LState) int {
		base := L.CheckString(2)
		ext := L.CheckString(3)
		L.Push(lua.LString(fmt.Sprintf("mock://global/%s.x64.%s", base, ext)))
		return 1
	}))

	// read_resource / read_global_resource / read_embed_resource -> string
	for _, name := range []string{"read_resource", "read_global_resource"} {
		vm.SetGlobal(name, vm.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LString("mock_content"))
			return 1
		}))
	}
	vm.SetGlobal("read_embed_resource", vm.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)
		if strings.HasPrefix(path, "embed://") {
			content, err := ReadEmbedResource(path)
			if err != nil {
				L.Push(lua.LString(""))
			} else {
				L.Push(lua.LString(string(content)))
			}
		} else {
			L.Push(lua.LString("mock_content"))
		}
		return 1
	}))

	// --- Session functions ---

	// active() -> mock session
	vm.SetGlobal("active", vm.NewFunction(func(L *lua.LState) int {
		session := &MockSession{
			Os: &MockOs{
				Arch:       "x64",
				Name:       "windows",
				Version:    "10.0.19041",
				ClrVersion: []string{"v4.0.30319", "v2.0.50727"},
			},
			IsAdmin:   true,
			SessionId: "mock-session-001",
		}
		L.Push(luar.New(L, session))
		return 1
	}))

	// isadmin(session) -> bool
	vm.SetGlobal("isadmin", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LTrue)
		return 1
	}))

	// --- BOF / execution functions ---

	// bof(session, resource, args, output) -> nil
	vm.SetGlobal("bof", vm.NewFunction(func(L *lua.LState) int {
		resource := ""
		if L.GetTop() >= 2 {
			resource = L.Get(2).String()
		}
		output := true
		if L.GetTop() >= 4 {
			output = lua.LVAsBool(L.Get(4))
		}
		h.mu.Lock()
		h.BofCalls = append(h.BofCalls, CapturedBofCall{
			Resource: resource,
			Output:   output,
		})
		h.mu.Unlock()
		L.Push(lua.LNil)
		return 1
	}))

	// execute_assembly / execute / execute_dll -> nil
	for _, name := range []string{"execute_assembly", "execute", "execute_dll"} {
		vm.SetGlobal(name, vm.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNil)
			return 1
		}))
	}

	// --- BOF packing functions ---

	// pack_bof_args(format, args_table) -> table
	vm.SetGlobal("pack_bof_args", vm.NewFunction(func(L *lua.LState) int {
		format := L.CheckString(1)
		argsTable := L.OptTable(2, L.NewTable())

		var args []string
		argsTable.ForEach(func(_, v lua.LValue) {
			args = append(args, v.String())
		})

		h.mu.Lock()
		h.BofPacks = append(h.BofPacks, CapturedBofPack{
			Format: format,
			Args:   args,
		})
		h.mu.Unlock()

		result := L.NewTable()
		result.Append(lua.LString("mock_packed"))
		L.Push(result)
		return 1
	}))

	// pack_bof(format, arg) -> string
	vm.SetGlobal("pack_bof", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("mock_packed"))
		return 1
	}))

	// pack_binary(data) -> string
	vm.SetGlobal("pack_binary", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("mock_packed"))
		return 1
	}))

	// --- Process / sacrifice functions ---

	// new_sacrifice(ppid, hidden, blockDll, disableETW, argue) -> table
	vm.SetGlobal("new_sacrifice", vm.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("Ppid", L.Get(1))
		t.RawSetString("Hidden", L.Get(2))
		L.Push(t)
		return 1
	}))

	// new_bypass(amsi, etw, wldp) -> table
	vm.SetGlobal("new_bypass", vm.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		if lua.LVAsBool(L.Get(1)) {
			t.RawSetString("bypass_amsi", lua.LString(""))
		}
		if lua.LVAsBool(L.Get(2)) {
			t.RawSetString("bypass_etw", lua.LString(""))
		}
		if lua.LVAsBool(L.Get(3)) {
			t.RawSetString("bypass_wldp", lua.LString(""))
		}
		L.Push(t)
		return 1
	}))

	// new_bypass_all() -> table
	vm.SetGlobal("new_bypass_all", vm.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("bypass_amsi", lua.LString(""))
		t.RawSetString("bypass_etw", lua.LString(""))
		t.RawSetString("bypass_wldp", lua.LString(""))
		L.Push(t)
		return 1
	}))

	// new_binary / new_64_executable / new_86_executable -> table
	for _, name := range []string{"new_binary", "new_64_executable", "new_86_executable"} {
		vm.SetGlobal(name, vm.NewFunction(func(L *lua.LState) int {
			L.Push(L.NewTable())
			return 1
		}))
	}

	// --- Task functions ---

	// wait / get / taskprint -> nil
	for _, name := range []string{"wait", "get", "taskprint"} {
		vm.SetGlobal(name, vm.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNil)
			return 1
		}))
	}
	vm.SetGlobal("assemblyprint", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	// --- Callback functions ---

	mockCallback := vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LTrue)
		return 1
	})
	for _, name := range []string{"callback_file", "callback_append", "callback_discard"} {
		fn := vm.NewFunction(func(L *lua.LState) int {
			L.Push(mockCallback)
			return 1
		})
		vm.SetGlobal(name, fn)
	}

	// --- Encode / utility functions ---

	vm.SetGlobal("base64_encode", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("mock_base64"))
		return 1
	}))
	vm.SetGlobal("base64_decode", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("mock_decoded"))
		return 1
	}))
	vm.SetGlobal("arg_hex", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("hex::mock"))
		return 1
	}))
	vm.SetGlobal("random_string", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("mockrandom"))
		return 1
	}))
	vm.SetGlobal("format_path", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(L.CheckString(1)))
		return 1
	}))
	vm.SetGlobal("file_exists", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LFalse)
		return 1
	}))
	vm.SetGlobal("timestamp", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("1234567890"))
		return 1
	}))
	vm.SetGlobal("timestamp_format", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("2024-01-01"))
		return 1
	}))
	vm.SetGlobal("shellsplit", vm.NewFunction(func(L *lua.LState) int {
		L.Push(L.NewTable())
		return 1
	}))
	vm.SetGlobal("is_full_path", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LFalse)
		return 1
	}))
	vm.SetGlobal("parse_octal", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(0))
		return 1
	}))
	vm.SetGlobal("parse_hex", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(0))
		return 1
	}))

	// --- Completer functions (no-op / mock return) ---

	// bind_flags_completer(cmd, completers_table) -> no return
	vm.SetGlobal("bind_flags_completer", vm.NewFunction(func(L *lua.LState) int {
		return 0
	}))

	// bind_args_completer(cmd, completers_table) -> no return
	vm.SetGlobal("bind_args_completer", vm.NewFunction(func(L *lua.LState) int {
		return 0
	}))

	// values_completer(values) -> nil (used as a completer object)
	vm.SetGlobal("values_completer", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))

	// artifact_name_completer() -> nil
	vm.SetGlobal("artifact_name_completer", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))

	// artifact_completer() -> nil
	vm.SetGlobal("artifact_completer", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))

	// rem_completer() -> nil
	vm.SetGlobal("rem_completer", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))

	// rem_agent_completer() -> nil
	vm.SetGlobal("rem_agent_completer", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))

	// --- Runtime functions (used inside command handlers, mocked for L4 testing) ---

	// execute_exe(session, resource, args, output, timeout, arch, process, sacrifice) -> nil
	vm.SetGlobal("execute_exe", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))

	// dllspawn(session, dll, export, shellcode, args, output, timeout, arch, process, sacrifice) -> nil
	vm.SetGlobal("dllspawn", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))

	// powershell(session, command, output) -> nil
	vm.SetGlobal("powershell", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))

	// self_stager(session) -> mock shellcode bytes
	vm.SetGlobal("self_stager", vm.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("mock_shellcode"))
		return 1
	}))

	// download_artifact(name, format, output) -> table, nil
	vm.SetGlobal("download_artifact", vm.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("bin", lua.LString("mock_artifact_bytes"))
		L.Push(t)
		L.Push(lua.LNil)
		return 2
	}))

	// --- UI annotation functions (no-op) ---

	for _, name := range []string{
		"ui_set", "ui_widget", "ui_group", "ui_placeholder",
		"ui_required", "ui_range", "ui_order",
	} {
		vm.SetGlobal(name, vm.NewFunction(func(L *lua.LState) int {
			return 0
		}))
	}

	// --- Plugin context globals ---

	vm.SetGlobal("plugin_dir", lua.LString("/mock/plugin"))
	vm.SetGlobal("plugin_resource_dir", lua.LString("/mock/plugin/resources"))
	vm.SetGlobal("plugin_name", lua.LString("community"))
	vm.SetGlobal("temp_dir", lua.LString("/tmp"))
	vm.SetGlobal("resource_dir", lua.LString("/mock/resources"))
}

// addEmbedLoader adds a custom Lua module loader that resolves require()
// calls against the embedded UnifiedFS. This supports the community plugin
// pattern: require("modules.lib") → community/community/modules/lib.lua
func (h *TestHarness) addEmbedLoader(vm *lua.LState) {
	loaders, ok := vm.GetField(vm.Get(lua.RegistryIndex), "_LOADERS").(*lua.LTable)
	if !ok {
		return
	}

	embedLoader := vm.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		luaPath := strings.Replace(name, ".", "/", -1) + ".lua"

		content, err := readCommunityFixture(luaPath)
		if err != nil {
			L.Push(lua.LString(fmt.Sprintf("no embedded module '%s'", name)))
			return 1
		}

		fn, err := L.LoadString(string(content))
		if err != nil {
			L.Push(lua.LString(fmt.Sprintf("error loading module '%s': %s", name, err.Error())))
			return 1
		}
		L.Push(fn)
		return 1
	})

	loaders.RawSetInt(loaders.Len()+1, embedLoader)
}

// preloadMockPackages registers mock versions of packages that Lua scripts
// load with require() but are normally provided by the server runtime.
func (h *TestHarness) preloadMockPackages(vm *lua.LState) {
	// Mock rpc package (used by rem.lua inside function bodies)
	vm.PreloadModule("rpc", func(L *lua.LState) int {
		mod := L.NewTable()
		L.Push(mod)
		return 1
	})

	// Mock beacon package
	vm.PreloadModule("beacon", func(L *lua.LState) int {
		mod := L.NewTable()
		L.Push(mod)
		return 1
	})
}
