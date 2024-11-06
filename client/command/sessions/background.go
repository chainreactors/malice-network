package sessions

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func BackGround(cmd *cobra.Command, con *repl.Console) error {
	con.ActiveTarget.Background()
	con.App.SwitchMenu(consts.ClientMenu)
	return nil
}
