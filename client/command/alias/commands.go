package alias

import (
	"github.com/chainreactors/grumble"
	completer "github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	aliasCmd := &grumble.Command{
		Name: consts.AliasStr,
		Help: "List current aliases",
		//LongHelp: help.GetHelpFor([]string{consts.AliasStr}),
		Run: func(ctx *grumble.Context) error {
			AliasesCmd(ctx, con)
			return nil
		},
		HelpGroup: consts.GenericGroup,
	}
	aliasCmd.AddCommand(&grumble.Command{
		Name: consts.AliasLoadStr,
		Help: "Load a command alias",
		//LongHelp: help.GetHelpFor([]string{consts.AliasesStr, consts.LoadStr}),
		Run: func(ctx *grumble.Context) error {
			AliasesLoadCmd(ctx, con)
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("dir-path", "path to the alias directory")
		},
		Completer: func(prefix string, args []string) []string {
			return completer.LocalPathCompleter(prefix, args, con)
		},
		HelpGroup: consts.GenericGroup,
	})

	aliasCmd.AddCommand(&grumble.Command{
		Name: consts.AliasInstallStr,
		Help: "Install a command alias",
		//LongHelp: help.GetHelpFor([]string{consts.AliasesStr, consts.InstallStr}),
		Run: func(ctx *grumble.Context) error {
			AliasesInstallCmd(ctx, con)
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the alias directory or tar.gz file")
		},
		Completer: func(prefix string, args []string) []string {
			return completer.LocalPathCompleter(prefix, args, con)
		},
		HelpGroup: consts.GenericGroup,
	})

	aliasCmd.AddCommand(&grumble.Command{
		Name: consts.AliasRemoveStr,
		Help: "Remove an alias",
		//LongHelp: help.GetHelpFor([]string{consts.RmStr}),
		Run: func(ctx *grumble.Context) error {
			AliasesRemoveCmd(ctx, con)
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("name", "name of the alias to remove")
		},
		Completer: func(prefix string, args []string) []string {
			return AliasCommandNameCompleter(prefix, args, con)
		},
		HelpGroup: consts.GenericGroup,
	})
	return []*grumble.Command{aliasCmd}
}
