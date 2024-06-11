package sys

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func WhoamiCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	sid := con.ActiveTarget.GetInteractive().SessionId
	if session == nil {
		return
	}
	whoamiTask, err := con.Rpc.Whoami(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleWhoami,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Whoami error: %v", err)
		return
	}
	con.AddCallback(whoamiTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		con.SessionLog(sid).Consolef("Username: %v\n", resp.GetOutput())
	})
}
