package basic

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

func removeCmd(ctx *grumble.Context, con *console.Console) {
	var id string
	if con.GetInteractive().SessionId != "" {
		id = con.GetInteractive().SessionId
	} else if ctx.Flags.String("id") != "" {
		id = ctx.Flags.String("id")
	} else {
		console.Log.Errorf("Require session id")
		return
	}
	_, err := con.Rpc.BasicSessionOP(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: id,
		IsDelete:  true,
	})
	if err != nil {
		logs.Log.Errorf("Session error: %v", err)
		return
	}
	con.UpdateSessions(false)
	session := con.Sessions[id]
	con.ActiveTarget.Set(session)
}
