package mal

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	cmd := &cobra.Command{
		Use:   consts.CommandMal,
		Short: "mal commands",
		//Long:  help.GetHelpFor(consts.CommandExtension),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalInstall,
		Short: "Install a mal manifest",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			MalInstallCmd(cmd, con)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalLoad,
		Short: "Load a mal manifest",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			MalLoadCmd(cmd, con)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalList,
		Short: "List mal manifests",
		Run: func(cmd *cobra.Command, args []string) {
			ListMalManiFest(con)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalRemove,
		Short: "Remove a mal manifest",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RemoveMalCmd(cmd, con)
		},
	})
	return []*cobra.Command{cmd}
}
