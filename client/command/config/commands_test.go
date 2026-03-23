package config

import (
	"strings"
	"testing"

	"github.com/chainreactors/malice-network/client/core"
)

func TestCommandsIncludeAIConfigSubcommand(t *testing.T) {
	cmds := Commands(&core.Console{})
	if len(cmds) != 1 {
		t.Fatalf("expected a single root config command, got %d", len(cmds))
	}

	root := cmds[0]
	aiCmd, _, err := root.Find([]string{"ai"})
	if err != nil {
		t.Fatalf("expected config ai command: %v", err)
	}
	if aiCmd == nil || aiCmd.Name() != "ai" {
		t.Fatalf("unexpected ai command: %#v", aiCmd)
	}
	if aiCmd.Hidden {
		t.Fatal("config ai command should be visible")
	}
	if !strings.Contains(aiCmd.Example, "config ai\n") {
		t.Fatalf("expected config ai examples, got:\n%s", aiCmd.Example)
	}
	if strings.Contains(aiCmd.Example, "ai-config --") {
		t.Fatalf("config ai examples should not advertise legacy alias, got:\n%s", aiCmd.Example)
	}
}
