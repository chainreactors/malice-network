package basic

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/console"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		{
			Name: "note",
			Help: "add note to session",
			Args: func(a *grumble.Args) {
				a.String("name", "session name")
			},
			Flags: func(f *grumble.Flags) {
				f.StringL("id", "", "session id")
			},
			Run: func(ctx *grumble.Context) error {
				noteCmd(ctx, con)
				return nil
			},
			Completer: func(prefix string, args []string) []string {
				if len(args) == 0 {
					return completer.BasicSessionIDCompleter(con, prefix)
				}
				return nil
			},
		},
		{
			Name: "group",
			Help: "group session",
			Args: func(a *grumble.Args) {
				a.String("group", "group name")
			},
			Flags: func(f *grumble.Flags) {
				f.StringL("id", "", "session id")
			},
			Run: func(ctx *grumble.Context) error {
				groupCmd(ctx, con)
				return nil
			},
			Completer: func(prefix string, args []string) []string {
				if len(args) == 0 {
					return completer.BasicSessionIDCompleter(con, prefix)
				}
				return nil
			},
		},
		{
			Name: "remove",
			Help: "remove session",
			Flags: func(f *grumble.Flags) {
				f.StringL("id", "", "session id")
			},
			Run: func(ctx *grumble.Context) error {
				removeCmd(ctx, con)
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
