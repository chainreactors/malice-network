package use

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
			Name:     "use",
			Help:     "Use session",
			LongHelp: help.GetHelpFor("use"),
			Args: func(a *grumble.Args) {
				a.String("sid", "session id")
			},
			Run: func(ctx *grumble.Context) error {
				UseSessionCmd(ctx, con)
				return nil
			},
			Completer: func(prefix string, args []string) []string {
				return completer.SessionIDCompleter(con, prefix)
			},
		},
		&grumble.Command{
			Name:     "background",
			LongHelp: help.GetHelpFor("background"),
			Help:     "back to root context",
			Run: func(ctx *grumble.Context) error {
				con.ActiveTarget.Background()
				return nil
			},
		},
	}
}

func UseSessionCmd(ctx *grumble.Context, con *console.Console) {
	var session *clientpb.Session
	err := con.UpdateSessions(false)
	if err != nil {
		console.Log.Errorf("%s", err)
		return
	}
	idArg := ctx.Args.String("sid")
	if idArg != "" {
		session = con.Sessions[idArg]
	}

	if session == nil {
		console.Log.Errorf(console.ErrNotFoundSession.Error())
		return
	}

	con.ActiveTarget.Set(session)
	con.EnableImplantCommands()
	console.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
}
