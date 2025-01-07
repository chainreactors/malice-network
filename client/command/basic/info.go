package basic

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func SessionInfoCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	if session == nil {
		return repl.ErrNotFoundSession
	}
	result := tui.RendStructDefault(session.Session, "Tasks")
	con.Log.Info("\n" + result)
	return nil
}
