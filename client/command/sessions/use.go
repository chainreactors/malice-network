package sessions

import (
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func UseSessionCmd(cmd *cobra.Command, con *repl.Console) error {
	var session *core.Session
	if session = con.GetSession(cmd.Flags().Arg(0)); session == nil {
		return repl.ErrNotFoundSession
	}
	session, err := con.UpdateSession(session.SessionId)
	if err != nil {
		return err
	}

	Use(con, session)
	return nil
}

func Use(con *repl.Console, sess *core.Session) {
	err := addon.RefreshAddonCommand(sess.Addons, con)
	if err != nil {
		core.Log.Errorf(err.Error())
		return
	}
	con.SwitchImplant(sess)
}
