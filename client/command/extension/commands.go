package extension

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	extensionCmd := &cobra.Command{
		Use:   consts.CommandExtension,
		Short: "Extension commands",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	extensionListCmd := &cobra.Command{
		Use:   consts.CommandExtensionList,
		Short: "List all extensions",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsCmd(cmd, con)
		},
	}

	extensionLoadCmd := &cobra.Command{
		Use:   consts.CommandExtensionLoad + " [extension]",
		Short: "Load an extension",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionLoadCmd(cmd, con)
		},
		Example: `
~~~
// Load an extension
extension load ./credman/
~~~
`,
	}

	common.BindArgCompletions(extensionLoadCmd, nil,
		carapace.ActionFiles().Usage("path to the extension directory"))

	extensionInstallCmd := &cobra.Command{
		Use:   consts.CommandExtensionInstall + " [extension_file]",
		Short: "Install an extension",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsInstallCmd(cmd, con)
		},
		Example: `
~~~
// Install an extension
extension install ./credman.tar.gz
~~~
`,
	}
	common.BindArgCompletions(extensionInstallCmd, nil,
		carapace.ActionFiles().Usage("path to the extension directory or tar.gz file"))

	extensionRemoveCmd := &cobra.Command{
		Use:   consts.CommandExtensionRemove + " [extension]",
		Short: "Remove an extension",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsRemoveCmd(cmd, con)
		},
		Example: `
~~~
// Remove an extension
extension remove credman
~~~
`,
	}

	common.BindArgCompletions(extensionRemoveCmd, nil,
		ExtensionsCommandNameCompleter(con).Usage("the command name of the extension to remove"))

	extensionCmd.AddCommand(extensionListCmd, extensionLoadCmd, extensionInstallCmd, extensionRemoveCmd)
	return []*cobra.Command{extensionCmd}
}

func Register(con *repl.Console) {
	for name, ext := range loadedExtensions {
		intermediate.RegisterInternalFunc(intermediate.ArmoryPackage, name, ext.Func, repl.WrapClientCallback(output.ParseBinaryResponse))
	}
}
