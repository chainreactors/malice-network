package sys

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func WhoamiCmd(cmd *cobra.Command, con *console.Console) {
	whoami(con)
}

func whoami(con *console.Console) {
	session := con.GetInteractive()
	sid := con.GetInteractive().SessionId
	if session == nil {
		return
	}
	whoamiTask, err := con.Rpc.Whoami(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleWhoami,
	})
	if err != nil {
		console.Log.Errorf("Whoami error: %v", err)
		return
	}
	con.AddCallback(whoamiTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		con.SessionLog(sid).Consolef("Username: %v\n", resp.GetOutput())
	})
}
