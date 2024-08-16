package modules

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		{
			Name:     consts.ModuleListModule,
			Help:     "list modules",
			LongHelp: help.GetHelpFor(consts.ModuleListModule),
			Run: func(ctx *grumble.Context) error {
				listModules(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		{
			Name:     consts.ModuleLoadModule,
			Help:     "load module",
			LongHelp: help.GetHelpFor(consts.ModuleLoadModule),
			Args: func(a *grumble.Args) {
				a.String("path", "path the module file")
			},
			Flags: func(f *grumble.Flags) {
				f.String("n", "name", "", "module name")
			},
			Run: func(ctx *grumble.Context) error {
				loadModule(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
			Completer: func(prefix string, args []string) []string {
				return completer.LocalPathCompleter(prefix, args, con)
			},
		},
	}
}
