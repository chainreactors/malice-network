package explorer

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	return []*cobra.Command{
		{
			Use:   consts.CommandExplore,
			Short: "file explorer",
			Run: func(cmd *cobra.Command, args []string) {
				explorerCmd(cmd, con)
				return
			},
		},
	}
}
