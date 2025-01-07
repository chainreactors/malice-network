package mal

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	cmd := &cobra.Command{
		Use:   consts.CommandMal,
		Short: "mal commands",
		//Long:  help.GetHelpFor(consts.CommandExtension),
		RunE: func(cmd *cobra.Command, args []string) error {
			return MalCmd(cmd, con)
		},
	}

	common.BindFlag(cmd, common.MalHttpFlagset)

	installCmd := &cobra.Command{
		Use:   consts.CommandMalInstall + " [mal_file]",
		Short: "Install a mal manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return MalInstallCmd(cmd, con)
		},
	}

	common.BindArgCompletions(installCmd,
		nil,
		carapace.ActionFiles().Usage("path the mal file to load"))

	cmd.AddCommand(installCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalLoad + " [mal]",
		Short: "Load a mal manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return MalLoadCmd(cmd, con)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalList,
		Short: "List mal manifests",
		Run: func(cmd *cobra.Command, args []string) {
			ListMalManifest(con)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalRemove + " [mal]",
		Short: "Remove a mal manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemoveMalCmd(cmd, con)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalRefresh,
		Short: "Refresh mal manifests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RefreshMalCmd(cmd, con)
		},
	})
	return []*cobra.Command{cmd}
}
