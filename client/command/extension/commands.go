package extension

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
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
		HelpGroup: "Extension",
	}
	extensionCmd.AddCommand(&grumble.Command{
		Name: "list",
		Help: "List all extensions",
		//LongHelp: help.GetHelpFor([]string{consts.CommandAlias}),
		Run: func(ctx *grumble.Context) error {
			ExtensionsListCmd(ctx, con)
			return nil
		},
		HelpGroup: "Extension",
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
		HelpGroup: "Extension",
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
		HelpGroup: "Extension",
	})
	return []*grumble.Command{extensionCmd}
}
