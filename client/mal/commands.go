package mal

import (
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *console.Console) []*cobra.Command {

	return []*cobra.Command{
		{
			Use:   "mal",
			Short: "mal commands",
			Long:  help.GetHelpFor(consts.CommandExtension),
			Run: func(cmd *cobra.Command, args []string) {
				MalLoadCmd(cmd, con)
			},
			GroupID: consts.GenericGroup,
		},
	}

}
