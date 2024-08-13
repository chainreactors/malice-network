package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func ChownCmd(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	path := ctx.Flags.String("path")
	uid := ctx.Flags.String("uid")
	gid := ctx.Flags.String("gid")
	recursive := ctx.Flags.Bool("recursive")
	chownTask, err := con.Rpc.Chown(con.ActiveTarget.Context(), &implantpb.ChownRequest{
		Path:      path,
		Uid:       uid,
		Gid:       gid,
		Recursive: recursive,
	})
	if err != nil {
		console.Log.Errorf("Chown error: %v", err)
		return
	}
	con.AddCallback(chownTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Response)
		con.SessionLog(sid).Consolef("Chown success\n")
	})
}
