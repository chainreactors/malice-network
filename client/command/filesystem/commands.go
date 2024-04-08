package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: consts.ModulePwd,
			Help: "Print working directory",
			//LongHelp: help.GetHelpFor([]string{consts.PwdStr}),
			Run: func(ctx *grumble.Context) error {
				PwdCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleCat,
			Help: "Print file content",
			Flags: func(f *grumble.Flags) {
				f.String("n", "name", "", "File name")
			},
			//LongHelp: help.GetHelpFor([]string{consts.CatStr}),
			Run: func(ctx *grumble.Context) error {
				CatCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleCd,
			Help: "Change directory",
			Flags: func(f *grumble.Flags) {
				f.String("p", "path", "", "Directory path")
			},
			//LongHelp: help.GetHelpFor([]string{consts.CdStr}),
			Run: func(ctx *grumble.Context) error {
				CdCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleChmod,
			Help: "Change file mode",
			Flags: func(f *grumble.Flags) {
				f.String("p", "path", "", "File path")
				f.String("m", "mode", "", "File mode")
			},
			//LongHelp: help.GetHelpFor([]string{consts.ChmodStr}),
			Run: func(ctx *grumble.Context) error {
				ChmodCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleChown,
			Help: "Change file owner",
			Flags: func(f *grumble.Flags) {
				f.String("p", "path", "", "File path")
				f.String("u", "uid", "", "User id")
				f.String("g", "gid", "", "Group id")
				f.Bool("r", "recursive", false, "Recursive")
			},
			//LongHelp: help.GetHelpFor([]string{consts.ChownStr}),
			Run: func(ctx *grumble.Context) error {
				ChownCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleCp,
			Help: "Copy file",
			Flags: func(f *grumble.Flags) {
				f.String("s", "source", "", "Source file")
				f.String("t", "target", "", "Target file")
			},
			//LongHelp: help.GetHelpFor([]string{consts.CpStr}),
			Run: func(ctx *grumble.Context) error {
				CpCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleLs,
			Help: "List directory",
			Flags: func(f *grumble.Flags) {
				f.String("p", "path", "", "Directory path")
			},
			//LongHelp: help.GetHelpFor([]string{consts.LsStr}),
			Run: func(ctx *grumble.Context) error {
				LsCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleMkdir,
			Help: "Make directory",
			Flags: func(f *grumble.Flags) {
				f.String("p", "path", "", "Directory path")
			},
			//LongHelp: help.GetHelpFor([]string{consts.MkdirStr}),
			Run: func(ctx *grumble.Context) error {
				MkdirCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleMv,
			Help: "Move file",
			Flags: func(f *grumble.Flags) {
				f.String("s", "source", "", "Source file")
				f.String("t", "target", "", "Target file")
			},
			//LongHelp: help.GetHelpFor([]string{consts.MvStr}),
			Run: func(ctx *grumble.Context) error {
				MvCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleRm,
			Help: "Remove file",
			Flags: func(f *grumble.Flags) {
				f.String("n", "name", "", "File name")
			},
			//LongHelp: help.GetHelpFor([]string{consts.RmStr}),
			Run: func(ctx *grumble.Context) error {
				RmCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
	}
}
