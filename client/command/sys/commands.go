package sys

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: consts.ModuleWhoami,
			Help: "Print current user",
			//LongHelp: help.GetHelpFor([]string{consts.WhoamiStr}),
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
			//LongHelp: help.GetHelpFor([]string{consts.UnameStr}),
			Run: func(ctx *grumble.Context) error {
				KillCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModulePs,
			Help: "List processes",
			//LongHelp: help.GetHelpFor([]string{consts.HostnameStr}),
			Run: func(ctx *grumble.Context) error {
				PsCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleEnv,
			Help: "List environment variables",
			//LongHelp: help.GetHelpFor([]string{consts.IdStr}),
			Run: func(ctx *grumble.Context) error {
				EnvCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleSetEnv,
			Help: "Set environment variable",
			Flags: func(f *grumble.Flags) {
				f.String("e", "env", "", "Environment variable")
				f.String("v", "value", "", "Value")
			},
			//LongHelp: help.GetHelpFor([]string{consts.IdStr}),
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
			//LongHelp: help.GetHelpFor([]string{consts.IdStr}),
			Run: func(ctx *grumble.Context) error {
				UnsetEnvCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
	}
}
