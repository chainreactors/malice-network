package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func CpCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	originPath := ctx.Flags.String("source")
	targetPath := ctx.Flags.String("target")
	args := []string{originPath, targetPath}
	mvTask, err := con.Rpc.Cp(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleCp,
		Args: args,
	})
	if err != nil {
		console.Log.Errorf("Cp error: %v", err)
		return
	}
	con.AddCallback(mvTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Cp success\n")
	})
}
