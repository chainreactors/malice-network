package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func MvCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	sourcePath := ctx.Flags.String("source")
	targetPath := ctx.Flags.String("target")
	args := []string{sourcePath, targetPath}
	mvTask, err := con.Rpc.Mv(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleMv,
		Args: args,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Mv error: %v", err)
		return
	}
	con.AddCallback(mvTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		console.Log.Consolef("Mv success\n")
	})
}
