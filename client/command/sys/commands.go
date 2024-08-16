package sys

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name:     consts.ModuleWhoami,
			Help:     "Print current user",
			LongHelp: help.GetHelpFor(consts.ModuleWhoami),
			Run: func(ctx *grumble.Context) error {
				WhoamiCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleKill,
			Help: "Kill the process",
			Flags: func(f *grumble.Flags) {
				f.String("p", "pid", "", "Process ID")
			},
			LongHelp: help.GetHelpFor(consts.ModuleKill),
			Run: func(ctx *grumble.Context) error {
				KillCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name:     consts.ModulePs,
			Help:     "List processes",
			LongHelp: help.GetHelpFor(consts.ModulePs),
			Run: func(ctx *grumble.Context) error {
				PsCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name:     consts.ModuleEnv,
			Help:     "List environment variables",
			LongHelp: help.GetHelpFor(consts.ModuleEnv),
			Run: func(ctx *grumble.Context) error {
				EnvCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name:     consts.ModuleSetEnv,
			Help:     "Set environment variable",
			LongHelp: help.GetHelpFor(consts.ModuleSetEnv),
			Flags: func(f *grumble.Flags) {
				f.String("e", "env", "", "Environment variable")
				f.String("v", "value", "", "Value")
			},
			Run: func(ctx *grumble.Context) error {
				SetEnvCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleUnsetEnv,
			Help: "Unset environment variable",
			Flags: func(f *grumble.Flags) {
				f.String("e", "env", "", "Environment variable")
			},
			LongHelp: help.GetHelpFor(consts.ModuleUnsetEnv),
			Run: func(ctx *grumble.Context) error {
				UnsetEnvCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name:     consts.ModuleNetstat,
			Help:     "List network connections",
			LongHelp: help.GetHelpFor(consts.ModuleNetstat),
			Run: func(ctx *grumble.Context) error {
				NetstatCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name:     consts.ModuleInfo,
			Help:     "get basic sys info",
			LongHelp: help.GetHelpFor(consts.ModuleInfo),
			Run: func(ctx *grumble.Context) error {
				InfoCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
	}
}
