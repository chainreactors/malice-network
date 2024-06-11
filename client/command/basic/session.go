package basic

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

func sessionCmd(ctx *grumble.Context, con *console.Console) {
	note := ctx.Flags.String("note")
	group := ctx.Flags.String("group")
	id := ctx.Args.String("id")
	isDelete := ctx.Flags.Bool("delete")
	_, err := con.Rpc.BasicSessionOP(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: id,
		Note:      note,
		GroupName: group,
		IsDelete:  isDelete,
	})
	if err != nil {
		logs.Log.Errorf("Session error: %v", err)
		return
	}
	con.UpdateSession()
}
