package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func PwdCmd(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	pwdTask, err := con.Rpc.Pwd(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModulePwd,
	})
	if err != nil {
		console.Log.Errorf("Pwd error: %v", err)
		return
	}
	con.AddCallback(pwdTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		con.SessionLog(sid).Consolef("%s\n", resp.GetOutput())
	})
}
