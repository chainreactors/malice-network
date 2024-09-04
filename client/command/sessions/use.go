package sessions

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func UseSessionCmd(cmd *cobra.Command, con *console.Console) {
	var session *clientpb.Session
	err := con.UpdateSessions(false)
	if err != nil {
		console.Log.Errorf("%s", err)
		return
	}
	idArg := cmd.Flags().Arg(0)
	if idArg != "" {
		session = con.Sessions[idArg]
	}

	if session == nil {
		console.Log.Errorf(console.ErrNotFoundSession.Error())
		return
	}

	con.SwitchImplant(session)
	console.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
}
