package modules

import (
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Commands(con *console.Console) []*cobra.Command {
	listModuleCmd := &cobra.Command{
		Use:   consts.ModuleListModule,
		Short: "List modules",
		Long:  help.GetHelpFor(consts.ModuleListModule),
		Run: func(cmd *cobra.Command, args []string) {
			ListModulesCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	loadModuleCmd := &cobra.Command{
		Use:   consts.ModuleLoadModule,
		Short: "Load module",
		Long:  help.GetHelpFor(consts.ModuleLoadModule),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			LoadModuleCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(loadModuleCmd).PositionalCompletion(
		carapace.ActionValues().Usage("module name"),
		carapace.ActionFiles().Usage("path to the module file"),
	)

	return []*cobra.Command{
		listModuleCmd,
		loadModuleCmd,
	}
}
