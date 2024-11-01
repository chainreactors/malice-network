package sessions

import (
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func NewSessionCmd(cmd *cobra.Command, con *repl.Console) error {

	return nil
}

func NewSession(con *repl.Console, sess *core.Session) {
	err := addon.RefreshAddonCommand(sess.Addons, con)
	if err != nil {
		core.Log.Errorf(err.Error())
		return
	}
	con.SwitchImplant(sess)
}
