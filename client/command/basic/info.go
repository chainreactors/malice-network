package basic

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func SessionInfoCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	if session == nil {
		return core.ErrNotFoundSession
	}
	result := tui.RendStructDefault(session.Session, "Tasks")
	con.Log.Info("\n" + result)
	return nil
}
