package explorer

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {

	regCommand := &cobra.Command{
		Use:   consts.CommandRegExplorer,
		Short: "registry explorer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return regExplorerCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend":   consts.ModuleRegListKey,
			"isStatic": "false",
		},
	}

	fileCmd := &cobra.Command{
		Use:   consts.CommandExplore,
		Short: "file explorer",
		Annotations: map[string]string{
			"isStatic": "false",
		},
		Run: func(cmd *cobra.Command, args []string) {
			fileExplorerCmd(cmd, con)
			return
		},
	}
	return []*cobra.Command{regCommand, fileCmd}
}
