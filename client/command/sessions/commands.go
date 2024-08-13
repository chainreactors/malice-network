package sessions

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/console"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: "sessions",
			Help: "List sessions",
			Flags: func(f *grumble.Flags) {
				f.String("i", "interact", "", "interact with a session")
				f.String("k", "kill", "", "kill the designated session")
				f.Bool("K", "kill-all", false, "kill all the sessions")
				f.Bool("C", "clean", false, "clean out any sessions marked as [DEAD]")
				f.Bool("F", "force", false, "force session action without waiting for results")
				f.Bool("a", "all", false, "show all sessions")
				//f.String("f", "filter", "", "filter sessions by substring")
				//f.String("e", "filter-re", "", "filter sessions by regular expression")

				f.Int("t", "timeout", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				SessionsCmd(ctx, con)
				return nil
			},
		},
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
