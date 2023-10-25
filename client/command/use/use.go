package use

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/desertbit/grumble"
	"strings"
)

func UseSessionCmd(ctx *grumble.Context, con *console.Console) {
	var session *clientpb.Session
	con.UpdateSession()
	idArg := ctx.Args.String("sid")
	if idArg != "" {
		session = con.Sessions[idArg]
	}

	if session == nil {
		console.Log.Errorf("session not found")
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
