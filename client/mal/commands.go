package mal

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {

	return []*grumble.Command{
		{
			Name: "mal",
			Help: "load mal plugin",
			//LongHelp: help.GetHelpFor(consts.ModuleListModule),
			Args: func(a *grumble.Args) {
				a.String("dir-path", "lua script")
			},
			Run: func(ctx *grumble.Context) error {
				MalLoadCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.GenericGroup,
		},
	}
}
