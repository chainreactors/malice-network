package file

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
			Name:     consts.ModuleDownload,
			Help:     "download file",
			LongHelp: help.GetHelpFor(consts.ModuleDownload),
			Flags: func(f *grumble.Flags) {
				f.String("n", "name", "", "filename")
				f.String("p", "path", "", "filepath")
			},
			Run: func(ctx *grumble.Context) error {
				download(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
		{
			Name:     consts.CommandSync,
			Help:     "sync file",
			LongHelp: help.GetHelpFor(consts.CommandSync),
			Flags: func(f *grumble.Flags) {
				f.String("i", "taskID", "", "task ID")
			},
			Run: func(ctx *grumble.Context) error {
				sync(ctx, con)
				return nil
			}, HelpGroup: consts.ImplantGroup,
		},
		{
			Name:     consts.ModuleUpload,
			Help:     "upload file",
			LongHelp: help.GetHelpFor(consts.ModuleUpload),
			Args: func(a *grumble.Args) {
				a.String("source", "file source path")
				a.String("destination", "target path")
			},
			Flags: func(f *grumble.Flags) {
				f.Int("", "priv", 0o644, "file Privilege")
				f.Bool("", "hidden", false, "filename")
			},
			Run: func(ctx *grumble.Context) error {
				upload(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
			Completer: func(prefix string, args []string) []string {
				if len(args) < 2 {
					return completer.LocalPathCompleter(prefix, args, con)
				}
				return nil
			}},
	}
}
