package explorer

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/spf13/cobra"
)

func Commands(con *console.Console) []*cobra.Command {
	return []*cobra.Command{
		{
			Use:   "explorer",
			Short: "file explorer",
			Run: func(cmd *cobra.Command, args []string) {
				explorerCmd(cmd, con)
				return
			},
		},
	}
}
