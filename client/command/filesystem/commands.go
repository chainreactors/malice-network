package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: consts.ModulePwd,
			Help: "Print working directory",
			//LongHelp: help.GetHelpFor([]string{consts.PwdStr}),
			Run: func(ctx *grumble.Context) error {
				PwdCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
	}
}
