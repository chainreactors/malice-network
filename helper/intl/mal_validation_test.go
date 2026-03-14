package intl

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

// newL4Harness creates a harness with a live VM for calling command handlers.
// The caller is responsible for closing the VM.
func newL4Harness(t *testing.T) (*TestHarness, *lua.LState) {
	t.Helper()
	requireCommunityFixture(t, "community/main.lua")
	h := NewTestHarness()
	vm := h.NewMockVM()

	if err := h.LoadCommunityMain(vm); err != nil {
		vm.Close()
		t.Fatalf("failed to load community main.lua: %v", err)
	}
	return h, vm
}

// callHandler is a helper that calls a command handler with the correct
// parameters based on the Lua function's prototype. It inspects DbgLocals
// to determine which parameters to pass (args table, cmd cobra.Command, etc.).
func callHandler(t *testing.T, h *TestHarness, vm *lua.LState, name string, args *lua.LTable) error {
	t.Helper()
	cmd, ok := h.Commands[name]
	if !ok {
		t.Skipf("command %q not registered", name)
		return nil
	}

	fn := cmd.LuaFunc
	numParams := int(fn.Proto.NumParameters)

	var luaArgs []lua.LValue
	for i := 0; i < numParams; i++ {
		paramName := fn.Proto.DbgLocals[i].Name
		switch paramName {
		case "args":
			luaArgs = append(luaArgs, args)
		case "cmd":
			luaArgs = append(luaArgs, luar.New(vm, cmd.CobraCmd))
		default:
			// Unknown parameter — push nil
			luaArgs = append(luaArgs, lua.LNil)
		}
	}

	return h.CallCommandHandler(vm, name, luaArgs...)
}

// TestReadfileRequiresFilepath verifies that the readfile command errors
// when no filepath is provided.
func TestReadfileRequiresFilepath(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	h, vm := newL4Harness(t)
	defer vm.Close()

	err := callHandler(t, h, vm, "readfile", vm.NewTable())
	if err == nil {
		t.Error("expected error for missing filepath, got nil")
		return
	}
	if !strings.Contains(err.Error(), "filepath is required") {
		t.Errorf("expected 'filepath is required' error, got: %v", err)
	}
}

// TestCurlRequiresHost verifies that the curl command errors when
// no host is provided.
func TestCurlRequiresHost(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	h, vm := newL4Harness(t)
	defer vm.Close()

	err := callHandler(t, h, vm, "curl", vm.NewTable())
	if err == nil {
		t.Error("expected error for missing host, got nil")
		return
	}
	if !strings.Contains(err.Error(), "host") {
		t.Errorf("expected error about host, got: %v", err)
	}
}

// TestTrustedpathRequiresDll verifies that the uac-bypass:trustedpath command
// errors when no DLL file is provided.
func TestTrustedpathRequiresDll(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	h, vm := newL4Harness(t)
	defer vm.Close()

	err := callHandler(t, h, vm, "uac-bypass:trustedpath", vm.NewTable())
	if err == nil {
		t.Error("expected error for missing local_dll_file, got nil")
		return
	}
	if !strings.Contains(err.Error(), "local_dll_file is required") {
		t.Errorf("expected 'local_dll_file is required' error, got: %v", err)
	}
}

// TestCmstpElevatedCOMRequiresArg verifies that uac-bypass:elevatedcom
// errors when no command argument is provided.
func TestCmstpElevatedCOMRequiresArg(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	h, vm := newL4Harness(t)
	defer vm.Close()

	err := callHandler(t, h, vm, "uac-bypass:elevatedcom", vm.NewTable())
	if err == nil {
		t.Error("expected error for missing command argument, got nil")
		return
	}
	if !strings.Contains(err.Error(), "Command argument required") {
		t.Errorf("expected 'Command argument required' error, got: %v", err)
	}
}

// TestValidArgsNoError verifies that commands succeed when given valid arguments.
func TestValidArgsNoError(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	h, vm := newL4Harness(t)
	defer vm.Close()

	// screenshot: takes no required args (uses session from active())
	t.Run("screenshot", func(t *testing.T) {
		err := callHandler(t, h, vm, "screenshot", vm.NewTable())
		if err != nil {
			t.Errorf("screenshot with no args should succeed, got: %v", err)
		}
	})

	// ipconfig: takes no required args
	t.Run("ipconfig", func(t *testing.T) {
		err := callHandler(t, h, vm, "ipconfig", vm.NewTable())
		if err != nil {
			t.Errorf("ipconfig with no args should succeed, got: %v", err)
		}
	})
}
