package sys

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func KillCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	sid := con.ActiveTarget.GetInteractive().SessionId
	if session == nil {
		return
	}
	pid := ctx.Flags.String("pid")
	killTask, err := con.Rpc.Kill(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleKill,
		Input: pid,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Kill error: %v", err)
		return
	}
	con.AddCallback(killTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Killed process")
	})
}
