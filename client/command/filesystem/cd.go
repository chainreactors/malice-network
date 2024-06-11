package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func CdCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	path := ctx.Flags.String("path")
	cdTask, err := con.Rpc.Cd(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleCd,
		Input: path,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Cd error: %v", err)
		return
	}
	con.AddCallback(cdTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite).GetResponse()
		con.SessionLog(sid).Consolef("Changed directory to: %s\n", path)
	})
}
