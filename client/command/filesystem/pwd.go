package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func PwdCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}

	pwdTask, err := con.Rpc.Pwd(con.ActiveTarget.Context(), &implantpb.Empty{})
	if err != nil {
		console.Log.Errorf("Pwd error: %v", err)
		return
	}
	con.AddCallback(pwdTask.TaskId, func(msg proto.Message) {
	})
}
