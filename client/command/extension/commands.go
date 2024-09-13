package extension

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
	extensionCmd := &cobra.Command{
		Use:   consts.CommandExtension,
		Short: "Extension commands",
		Long:  help.GetHelpFor(consts.CommandExtension),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	extensionListCmd := &cobra.Command{
		Use:   consts.CommandExtensionList,
		Short: "List all extensions",
		Long:  help.GetHelpFor(consts.CommandExtension + " " + consts.CommandExtensionList),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsCmd(cmd, con)
		},
	}

	extensionLoadCmd := &cobra.Command{
		Use:   consts.CommandExtensionLoad,
		Short: "Load an extension",
		Long:  help.GetHelpFor(consts.CommandExtension + " " + consts.CommandExtensionLoad),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionLoadCmd(cmd, con)
		},
	}

	common.BindArgCompletions(extensionLoadCmd, nil,
		carapace.ActionFiles().Usage("path to the extension directory"))

	extensionInstallCmd := &cobra.Command{
		Use:   consts.CommandExtensionInstall,
		Short: "Install an extension",
		Long:  help.GetHelpFor(consts.CommandExtension + " " + consts.CommandExtensionInstall),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsInstallCmd(cmd, con)
		},
	}
	common.BindArgCompletions(extensionInstallCmd, nil,
		carapace.ActionFiles().Usage("path to the extension directory or tar.gz file"))

	extensionRemoveCmd := &cobra.Command{
		Use:   consts.CommandExtensionRemove,
		Short: "Remove an extension",
		Long:  help.GetHelpFor(consts.CommandExtension + " " + consts.CommandExtensionRemove),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsRemoveCmd(cmd, con)
		},
	}

	common.BindArgCompletions(extensionRemoveCmd, nil,
		ExtensionsCommandNameCompleter(con).Usage("the command name of the extension to remove"))

	extensionCmd.AddCommand(extensionListCmd, extensionLoadCmd, extensionInstallCmd, extensionRemoveCmd)
	return []*cobra.Command{extensionCmd}
}

func Register(con *repl.Console) {
	for name, ext := range loadedExtensions {
		intermediate.RegisterInternalFunc(name, ext.Func)
	}
}
