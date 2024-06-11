package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func MkdirCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	path := ctx.Flags.String("path")
	mkdirTask, err := con.Rpc.Mkdir(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleMkdir,
		Input: path,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Mkdir error: %v", err)
		return
	}
	con.AddCallback(mkdirTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Created directory\n")
	})
}
