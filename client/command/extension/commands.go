package extension

import (
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Commands(con *console.Console) []*cobra.Command {
	extensionCmd := &cobra.Command{
		Use:   "extension",
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
	carapace.Gen(extensionLoadCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path to the extension directory"),
	)

	extensionInstallCmd := &cobra.Command{
		Use:   consts.CommandExtensionInstall,
		Short: "Install an extension",
		Long:  help.GetHelpFor(consts.CommandExtension + " " + consts.CommandExtensionInstall),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsInstallCmd(cmd, con)
		},
	}
	carapace.Gen(extensionInstallCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path to the extension directory or tar.gz file"),
	)

	extensionRemoveCmd := &cobra.Command{
		Use:   consts.CommandExtensionRemove,
		Short: "Remove an extension",
		Long:  help.GetHelpFor(consts.CommandExtension + " " + consts.CommandExtensionRemove),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsRemoveCmd(cmd, con)
		},
	}
	carapace.Gen(extensionRemoveCmd).PositionalCompletion(ExtensionsCommandNameCompleter(con).Usage("the command name of the extension to remove"))

	extensionCmd.AddCommand(extensionListCmd, extensionLoadCmd, extensionInstallCmd, extensionRemoveCmd)
	return []*cobra.Command{extensionCmd}
}
