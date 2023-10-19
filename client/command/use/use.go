package use

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/desertbit/grumble"
	"strings"
)

func UseSessionCmd(ctx *grumble.Context, con *console.Console) {

	var session *clientpb.Session
	var err error
	idArg := ctx.Args.String("id")
	if idArg != "" {
		session = core.Sessions[idArg]
	}

	if session != nil {
		logs.Log.Errorf("session not found", err)
		return
	}

	con.ActiveTarget.Set(session)
	logs.Log.Infof("Active session %s (%s)\n", session.Name, session.SessionId)

}

func SessionIDCompleter(prefix string) (results []string) {
	for _, s := range core.Sessions {
		if strings.HasPrefix(s.SessionId, prefix) {
			results = append(results, s.SessionId)
		}
	}
	return
}
