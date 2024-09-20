package sessions

import (
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func UseSessionCmd(cmd *cobra.Command, con *repl.Console) {
	var session *core.Session
	if session = con.GetSession(cmd.Flags().Arg(0)); session == nil {
		con.Log.Errorf(repl.ErrNotFoundSession.Error())
		return
	}
	session, err := con.UpdateSession(session.SessionId)
	if err != nil {
		con.Log.Errorf(err.Error())
	}

	Use(con, session)
	con.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
}

func Use(con *repl.Console, sess *core.Session) {
	err := addon.RefreshAddonCommand(sess.Addons.Addons, con)
	if err != nil {
		core.Log.Errorf(err.Error())
		return
	}
	con.SwitchImplant(sess)
}
