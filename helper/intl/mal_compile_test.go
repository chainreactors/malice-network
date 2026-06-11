package intl

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

// TestAllLuaFilesCompile verifies that every .lua file in UnifiedFS
// can be parsed and compiled without errors (L1 syntax verification).
func TestAllLuaFilesCompile(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	files, err := GetAllLuaFiles()
	if err != nil {
		t.Fatalf("failed to enumerate lua files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no .lua files found in UnifiedFS")
	}

	t.Logf("found %d .lua files", len(files))

	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			content, err := UnifiedFS.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			reader := strings.NewReader(string(content))
			chunk, err := parse.Parse(reader, path)
			if err != nil {
				t.Fatalf("parse error in %s: %v", path, err)
			}

			_, err = lua.Compile(chunk, path)
			if err != nil {
				t.Fatalf("compile error in %s: %v", path, err)
			}
		})
	}
}

// TestCommunityMainLoads verifies that the community main.lua entry point
// and all its required modules can be loaded and executed in the mock VM
// without errors. This is the key integration test for the harness.
func TestCommunityMainLoads(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	harness := NewTestHarness()
	vm := harness.NewMockVM()
	defer vm.Close()

	err := harness.LoadCommunityMain(vm)
	if err != nil {
		t.Fatalf("failed to load community main.lua: %v", err)
	}

	if len(harness.Commands) == 0 {
		t.Fatal("no commands were registered after loading main.lua")
	}

	t.Logf("successfully registered %d commands", len(harness.Commands))
}
