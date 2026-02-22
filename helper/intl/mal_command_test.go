package intl

import (
	"testing"
)

// sharedHarness caches the result of loading community main.lua once.
// Multiple tests can reuse this without re-executing the Lua scripts.
var sharedHarness *TestHarness

func getSharedHarness(t *testing.T) *TestHarness {
	t.Helper()
	if sharedHarness != nil {
		return sharedHarness
	}
	sharedHarness = NewTestHarness()
	vm := sharedHarness.NewMockVM()
	defer vm.Close()

	if err := sharedHarness.LoadCommunityMain(vm); err != nil {
		t.Fatalf("failed to load community main.lua: %v", err)
	}
	return sharedHarness
}

// TestCommandCount verifies that a reasonable number of commands are registered.
func TestCommandCount(t *testing.T) {
	h := getSharedHarness(t)

	count := len(h.Commands)
	t.Logf("total commands registered: %d", count)

	if count < 50 {
		t.Errorf("expected at least 50 commands, got %d", count)
	}
}

// TestKeyCommandsExist verifies that specific important commands are registered.
func TestKeyCommandsExist(t *testing.T) {
	h := getSharedHarness(t)

	required := []string{
		"screenshot",
		"curl",
		"ipconfig",
		"mimikatz",
		"nanodump",
		"ldapsearch",
		"move:psexec",
		"move:dcom",
		"bof-execute_assembly",
		"load_prebuild",
		"persistence:reg_key",
		"token:make",
		"token:steal",
		"route:print",
	}

	for _, name := range required {
		t.Run(name, func(t *testing.T) {
			if _, ok := h.Commands[name]; !ok {
				t.Errorf("expected command %q to be registered", name)
			}
		})
	}
}

// TestAllCommandsHaveTTP verifies that every registered command has a
// non-empty TTP annotation. Commands that are utility/admin commands
// may intentionally lack a TTP and are whitelisted.
func TestAllCommandsHaveTTP(t *testing.T) {
	h := getSharedHarness(t)

	// Commands that intentionally don't map to a MITRE TTP
	whitelist := map[string]bool{
		"load_prebuild":         true,
		"rem_community:load":    true,
		"rem_community:socks5":  true,
		"rem_community:connect": true,
		"rem_community:fork":    true,
		"rem_community:run":     true,
		"rem_community:log":     true,
		"rem_community:stop":    true,
		"persistence:Junction_Folder": true,
	}

	missing := 0
	for name, cmd := range h.Commands {
		t.Run(name, func(t *testing.T) {
			if cmd.TTP == "" && !whitelist[name] {
				t.Errorf("command %q has no TTP annotation", name)
				missing++
			}
		})
	}
}

// TestAllCommandsHaveShortDescription verifies that every registered command
// has a non-empty short description.
func TestAllCommandsHaveShortDescription(t *testing.T) {
	h := getSharedHarness(t)

	for name, cmd := range h.Commands {
		t.Run(name, func(t *testing.T) {
			if cmd.Short == "" {
				t.Errorf("command %q has no short description", name)
			}
		})
	}
}

// TestAllCommandsHaveOpsec verifies that commands have an OPSEC score set.
// Some utility commands may intentionally skip opsec, which are whitelisted.
func TestAllCommandsHaveOpsec(t *testing.T) {
	h := getSharedHarness(t)

	// Commands that intentionally don't set an OPSEC score
	whitelist := map[string]bool{
		"load_prebuild": true,
	}

	missing := 0
	for name, cmd := range h.Commands {
		t.Run(name, func(t *testing.T) {
			if !cmd.HasOpsec && !whitelist[name] {
				t.Logf("command %q has no opsec score", name)
				missing++
			}
		})
	}

	if missing > 0 {
		t.Logf("%d commands lack opsec scores (non-fatal)", missing)
	}
}

// TestOpsecScoresInRange verifies that OPSEC scores are in a reasonable range (0-10).
func TestOpsecScoresInRange(t *testing.T) {
	h := getSharedHarness(t)

	for name, cmd := range h.Commands {
		if !cmd.HasOpsec {
			continue
		}
		t.Run(name, func(t *testing.T) {
			if cmd.OpsecScore < 0 || cmd.OpsecScore > 10 {
				t.Errorf("command %q has opsec score %.1f outside range [0, 10]", name, cmd.OpsecScore)
			}
		})
	}
}

// TestKeyCommandFlags verifies that specific commands have their expected flags
// registered on the cobra.Command.
func TestKeyCommandFlags(t *testing.T) {
	h := getSharedHarness(t)

	tests := []struct {
		cmd   string
		flags []string
	}{
		{"curl", []string{"host", "method", "body", "header"}},
		{"nanodump", []string{"pid", "write"}},
		{"ldapsearch", []string{"query", "attributes"}},
		{"move:psexec", []string{"host", "service", "path"}},
		{"bof-execute_assembly", []string{"amsi", "etw", "patchexit"}},
		{"elevate:EfsPotato", []string{"command", "shellcode-file", "shellcode-artifact"}},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			cmd, ok := h.Commands[tt.cmd]
			if !ok {
				t.Skipf("command %q not registered", tt.cmd)
				return
			}

			for _, flag := range tt.flags {
				if cmd.CobraCmd.Flags().Lookup(flag) == nil {
					t.Errorf("command %q missing expected flag %q", tt.cmd, flag)
				}
			}
		})
	}
}

// TestAllCommandsHaveLuaFunc verifies that every registered command has a
// valid Lua function handler reference.
func TestAllCommandsHaveLuaFunc(t *testing.T) {
	h := getSharedHarness(t)

	for name, cmd := range h.Commands {
		t.Run(name, func(t *testing.T) {
			if cmd.LuaFunc == nil {
				t.Errorf("command %q has nil LuaFunc", name)
			}
		})
	}
}
