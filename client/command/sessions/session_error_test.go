package sessions

import (
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	clientcore "github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func TestNoteCmdRequiresSessionSelection(t *testing.T) {
	con := newBareSessionConsole()
	cmd := &cobra.Command{Use: "note"}

	if err := noteCmd(cmd, con); err == nil {
		t.Fatal("expected noteCmd to fail when no session is selected")
	}
}

func TestGroupCmdRequiresSessionSelection(t *testing.T) {
	con := newBareSessionConsole()
	cmd := &cobra.Command{Use: "group"}

	if err := groupCmd(cmd, con); err == nil {
		t.Fatal("expected groupCmd to fail when no session is selected")
	}
}

func newBareSessionConsole() *clientcore.Console {
	return &clientcore.Console{
		Server: &clientcore.Server{
			ServerState: &iomclient.ServerState{
				ActiveTarget: &iomclient.ActiveTarget{},
			},
		},
		Log: iomclient.Log,
	}
}
