package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func RmCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	fileName := ctx.Flags.String("name")
	rmTask, err := con.Rpc.Rm(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleRm,
		Input: fileName,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Rm error: %v", err)
		return
	}
	con.AddCallback(rmTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Removed file success\n")
	})
}
