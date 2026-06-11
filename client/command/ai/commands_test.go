package ai

import (
	"testing"

	"github.com/chainreactors/malice-network/client/core"
)

func TestRootAICommandsDoNotExposeLegacyConfigAlias(t *testing.T) {
	cmds := Commands(&core.Console{})

	for _, cmd := range cmds {
		if cmd.Name() == "ai-config" {
			t.Fatalf("legacy ai-config command should not be registered: %#v", cmd)
		}
	}
}
