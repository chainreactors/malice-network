package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/console"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		{
			Name: "download",
			Help: "download file",
			Flags: func(f *grumble.Flags) {
				f.String("n", "name", "", "filename")
				f.String("p", "path", "", "filepath")
			},
			Run: func(ctx *grumble.Context) error {
				download(ctx, con)
				return nil
			},
		},
		{
			Name: "sync",
			Help: "sync file",
			Flags: func(f *grumble.Flags) {
				f.String("n", "name", "", "filename")
			},
			Run: func(ctx *grumble.Context) error {
				sync(ctx, con)
				return nil
			},
			Completer: func(prefix string, args []string) []string {
				return completer.SessionIDCompleter(con, prefix)
			},
		},
		{
			Name: "upload",
			Help: "upload file",
			Flags: func(f *grumble.Flags) {
				f.String("n", "name", "", "filename")
				f.String("p", "path", "", "filepath")
				f.String("t", "target", "", "file in implant target")
				f.Int("", "priv", 0o644, "file Privilege")
				f.Bool("", "hidden", false, "filename")
			},
			Run: func(ctx *grumble.Context) error {
				upload(ctx, con)
				return nil
			},
		},
	}
}
