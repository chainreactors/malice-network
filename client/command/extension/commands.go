package extension

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	extensionCmd := &grumble.Command{
		Name: "extension",
		Help: "Extension commands",
		//LongHelp: help.GetHelpFor([]string{consts.CommandAlias}),
		Run: func(ctx *grumble.Context) error {
			ExtensionsCmd(ctx, con)
			return nil
		},
		HelpGroup: consts.GenericGroup,
	}
	extensionCmd.AddCommand(&grumble.Command{
		Name: consts.CommandExtensionList,
		Help: "List all extensions",
		//LongHelp: help.GetHelpFor([]string{consts.CommandAlias}),
		Run: func(ctx *grumble.Context) error {
			ExtensionsListCmd(ctx, con)
			return nil
		},
		HelpGroup: "Extension",
	})
	extensionCmd.AddCommand(&grumble.Command{
		Name: consts.CommandExtensionLoad,
		Help: "Load an extension",
		Args: func(a *grumble.Args) {
			a.String("dir-path", "path to the extension directory")
		},
		//LongHelp: help.GetHelpFor([]string{consts.CommandAlias}),
		Run: func(ctx *grumble.Context) error {
			ExtensionLoadCmd(ctx, con)
			return nil
		},
	})
	extensionCmd.AddCommand(&grumble.Command{
		Name: "install",
		Help: "Install an extension",
		//LongHelp: help.GetHelpFor([]string{consts.CommandAlias}),
		Run: func(ctx *grumble.Context) error {
			ExtensionsInstallCmd(ctx, con)
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the extension directory or tar.gz file")
		},
	})
	extensionCmd.AddCommand(&grumble.Command{
		Name: "remove",
		Help: "Remove an extension",
		//LongHelp: help.GetHelpFor([]string{consts.CommandAlias}),
		Run: func(ctx *grumble.Context) error {
			ExtensionsRemoveCmd(ctx, con)
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("name", "name of the extension to remove")
		},
	})
	return []*grumble.Command{extensionCmd}
}
