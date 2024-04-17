package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		{
			Name: consts.ModuleDownload,
			Help: "download file",
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
			Name: consts.CommandSync,
			Help: "sync file",
			Flags: func(f *grumble.Flags) {
				f.String("i", "taskID", "", "task ID")
			},
			Run: func(ctx *grumble.Context) error {
				sync(ctx, con)
				return nil
			}, HelpGroup: consts.ImplantGroup,
		},
		{
			Name: consts.ModuleUpload,
			Help: "upload file",
			Flags: func(f *grumble.Flags) {
				f.String("s", "source", "", "source path")
				f.String("d", "destination", "", "target path")
				f.Int("", "priv", 0o644, "file Privilege")
				f.Bool("", "hidden", false, "filename")
			},
			Run: func(ctx *grumble.Context) error {
				upload(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
	}
}
