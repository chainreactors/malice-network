package sessions

import (
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func UseSessionCmd(cmd *cobra.Command, con *repl.Console) error {
	var session *core.Session
	sid := cmd.Flags().Arg(0)
	var err error
	if session = con.GetSession(sid); session == nil {
		session, err = con.UpdateSession(sid)
		if err != nil {
			return err
		}
	}

	return Use(con, session)
}

func Use(con *repl.Console, sess *core.Session) error {
	err := addon.RefreshAddonCommand(sess.Addons, con)
	if err != nil {
		return err
	}
	con.SwitchImplant(sess)
	return nil
}
