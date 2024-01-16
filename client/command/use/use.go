package use

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"strings"
)

func Command(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: "use",
			Help: "Use session",
			Args: func(a *grumble.Args) {
				a.String("sid", "session id")
			},
			Run: func(ctx *grumble.Context) error {
				UseSessionCmd(ctx, con)
				return nil
			},
			Completer: func(prefix string, args []string) []string {
				return SessionIDCompleter(con, prefix)
			},
		},
		&grumble.Command{
			Name: "back",
			Help: "back to root context",
			Run: func(ctx *grumble.Context) error {
				con.ActiveTarget.Set(nil)
				return nil
			},
		},
	}
}

func UseSessionCmd(ctx *grumble.Context, con *console.Console) {
	var session *clientpb.Session
	con.UpdateSession()
	idArg := ctx.Args.String("sid")
	if idArg != "" {
		session = con.Sessions[idArg]
	}

	if session == nil {
		console.Log.Errorf(console.ErrNotFoundSession.Error())
		return
	}

	con.ActiveTarget.Set(session)
	console.Log.Infof("Active session %s (%s)\n", session.Name, session.SessionId)
}

func SessionIDCompleter(con *console.Console, prefix string) (results []string) {
	for _, s := range con.Sessions {
		if strings.HasPrefix(s.SessionId, prefix) {
			results = append(results, s.SessionId)
		}
	}
	return
}
