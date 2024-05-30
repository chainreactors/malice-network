package modules

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		{
			Name: consts.ModuleListModule,
			Help: "list modules",
			Run: func(ctx *grumble.Context) error {
				listModules(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		{
			Name: consts.ModuleLoadModule,
			Help: "load module",
			Flags: func(f *grumble.Flags) {
				f.String("n", "name", "", "module name")
				f.String("p", "path", "", "module path")
			},
			Run: func(ctx *grumble.Context) error {
				loadModule(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
	}
}
