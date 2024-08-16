package observe

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

func Command(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name:     "observe",
			Help:     "observe session",
			LongHelp: help.GetHelpFor("observe"),
			Args: func(a *grumble.Args) {
				a.StringList("sid", "session id")
			},
			Flags: func(f *grumble.Flags) {
				f.Bool("r", "remove", false, "remove observe")
				f.Bool("l", "list", false, "list all observers")
			},
			Run: func(ctx *grumble.Context) error {
				ObserveCmd(ctx, con)
				return nil
			},
			Completer: func(prefix string, args []string) []string {
				return completer.SessionIDCompleter(con, prefix)
			},
		},
	}
}

func ObserveCmd(ctx *grumble.Context, con *console.Console) {
	var session *clientpb.Session
	if ctx.Flags.Bool("list") {
		for i, ob := range con.Observers {
			console.Log.Infof("%d: %s", i, ob.SessionId())
		}
		return
	}

	idArg := ctx.Args.StringList("sid")
	if idArg == nil {
		if con.GetInteractive() != nil {
			idArg = []string{con.GetInteractive().SessionId}
		} else {
			for i, ob := range con.Observers {
				console.Log.Infof("%d: %s", i, ob.SessionId())
			}
			return
		}
	}
	for _, sid := range idArg {
		session = con.Sessions[sid]

		if session == nil {
			console.Log.Warn(console.ErrNotFoundSession.Error())
		}

		if ctx.Flags.Bool("remove") {
			con.RemoveObserver(session.SessionId)
		} else {
			con.AddObserver(session)
		}
	}
}
