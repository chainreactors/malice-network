package pipeline

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"
)

func TestCommandsExposeExpectedPipelineRoots(t *testing.T) {
	cmds := Commands(&core.Console{})
	if len(cmds) != 5 {
		t.Fatalf("pipeline command roots = %d, want 5", len(cmds))
	}

	want := map[string]bool{
		consts.CommandPipelineTcp:  true,
		consts.HTTPPipeline:        true,
		consts.CommandPipelineBind: true,
		consts.CommandRem:          true,
		"webshell":                 true,
	}
	for _, cmd := range cmds {
		delete(want, cmd.Name())
	}
	if len(want) != 0 {
		t.Fatalf("missing pipeline roots: %#v", want)
	}
}

func TestCommandsExposeRemUpdateIntervalSubcommand(t *testing.T) {
	var remCmdName string
	for _, cmd := range Commands(&core.Console{}) {
		if cmd.Name() == consts.CommandRem {
			remCmdName = cmd.Name()
			updateCmd, _, err := cmd.Find([]string{"update", "interval"})
			if err != nil {
				t.Fatalf("expected rem update interval command under %s: %v", remCmdName, err)
			}
			if updateCmd == nil || updateCmd.Name() != "interval" {
				t.Fatalf("unexpected rem update interval command: %#v", updateCmd)
			}
			return
		}
	}
	t.Fatalf("rem command %q not found", consts.CommandRem)
}
