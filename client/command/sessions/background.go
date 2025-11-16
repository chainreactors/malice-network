package sessions

import (
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func BackGround(cmd *cobra.Command, con *core.Console) error {
	con.ActiveTarget.Background()
	con.App.SwitchMenu(consts.ClientMenu)
	return nil
}
