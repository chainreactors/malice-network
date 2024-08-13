package sessions

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

func groupCmd(ctx *grumble.Context, con *console.Console) {
	group := ctx.Args.String("group")
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
		GroupName: group,
	})
	if err != nil {
		logs.Log.Errorf("Session error: %v", err)
		return
	}
	session := con.Sessions[id]
	con.ActiveTarget.Set(session)
}
