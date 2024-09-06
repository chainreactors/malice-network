package sessions

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func UseSessionCmd(cmd *cobra.Command, con *repl.Console) {
	var session *repl.Session
	if session = con.Sessions[cmd.Flags().Arg(0)]; session == nil {
		repl.Log.Errorf(repl.ErrNotFoundSession.Error())
		return
	}

	con.SwitchImplant(session)
	repl.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
}
