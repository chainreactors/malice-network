package explorer

import (
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func Commands(con *core.Console) []*cobra.Command {

	regCommand := &cobra.Command{
		Use:   consts.CommandRegExplorer + " [hive\\path]",
		Short: "Interactive registry explorer",
		Long:  "Explore registry keys and values interactively from a starting hive/path (e.g., HKEY_LOCAL_MACHINE\\SOFTWARE).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return regExplorerCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend":     consts.ModuleRegListKey,
			"thirdParty": "true",
		},
		Example: `~~~
reg_explorer HKLM\\SOFTWARE
reg_explorer HKEY_CURRENT_USER\\Software
~~~`,
	}

	fileCmd := &cobra.Command{
		Use:   consts.CommandExplore,
		Short: "file explorer",
		Annotations: map[string]string{
			"thirdParty": "true",
		},
		Run: func(cmd *cobra.Command, args []string) {
			fileExplorerCmd(cmd, con)
			return
		},
	}
	return []*cobra.Command{regCommand, fileCmd}
}
