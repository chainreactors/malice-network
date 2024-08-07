package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func CatCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	fileName := ctx.Flags.String("name")
	catTask, err := con.Rpc.Cat(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleCat,
		Input: fileName,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Cat error: %v", err)
		return
	}
	con.AddCallback(catTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		console.Log.Consolef("File content: %s\n", resp.GetOutput())
	})
}
