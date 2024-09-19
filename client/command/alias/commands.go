package alias

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	aliasCmd := &cobra.Command{
		Use:   consts.CommandAlias,
		Short: "manage aliases",
		Long:  help.GetHelpFor(consts.CommandAlias),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			return
		},
	}

	aliasListCmd := &cobra.Command{
		Use:   consts.CommandAliasList,
		Short: "List all aliases",
		Long:  help.GetHelpFor(consts.CommandAlias + " " + consts.CommandAliasList),
		Run: func(cmd *cobra.Command, args []string) {
			AliasesCmd(cmd, con)
			return
		},
	}

	aliasLoadCmd := &cobra.Command{
		Use:   consts.CommandAliasLoad + " [alias]",
		Short: "Load a command alias",
		Long:  help.GetHelpFor(consts.CommandAlias + " " + consts.CommandAliasLoad),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			AliasesLoadCmd(cmd, con)
			return
		},
	}
	common.BindArgCompletions(
		aliasLoadCmd,
		nil,
		carapace.ActionFiles().Usage("local path where the downloaded file will be saved (optional)"),
	)

	aliasInstallCmd := &cobra.Command{
		Use:   consts.CommandAliasInstall + " [alias_file]",
		Short: "Install a command alias",
		Long:  help.GetHelpFor(consts.CommandAlias + " " + consts.CommandAliasInstall),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			AliasesInstallCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(aliasInstallCmd,
		nil,
		carapace.ActionFiles().Usage("local path where the downloaded file will be saved (optional)"),
	)

	aliasRemoveCmd := &cobra.Command{
		Use:   consts.CommandAliasRemove + " [alias]",
		Short: "Remove an alias",
		Long:  help.GetHelpFor(consts.CommandAlias + " " + consts.CommandAliasRemove),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			AliasesRemoveCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(
		aliasRemoveCmd,
		nil,
		AliasCompleter())

	aliasCmd.AddCommand(aliasListCmd, aliasLoadCmd, aliasInstallCmd, aliasRemoveCmd)
	return []*cobra.Command{aliasCmd}

}

func Register(con *repl.Console) {
	for name, aliasPkg := range loadedAliases {
		intermediate.RegisterInternalFunc(name, aliasPkg.Func, repl.WrapImplantCallback(common.ParseAssembly))
	}
}
