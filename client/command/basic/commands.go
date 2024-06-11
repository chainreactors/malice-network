package basic

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/console"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		{
			Name: "session",
			Help: "session operations",
			Args: func(a *grumble.Args) {
				a.String("id", "session id")
			},
			Flags: func(f *grumble.Flags) {
				f.String("n", "note", "", "session note")
				f.String("g", "group", "", "session group")
				f.Bool("d", "delete", false, "delete session")
			},
			Run: func(ctx *grumble.Context) error {
				sessionCmd(ctx, con)
				return nil
			},
			Completer: func(prefix string, args []string) []string {
				if len(args) == 0 {
					return completer.BasicSessionIDCompleter(con, prefix)
				}
				return nil
			},
		},
	}
}
