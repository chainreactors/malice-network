package sys

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func InfoCmd(cmd *cobra.Command, con *console.Console) {
	info(con)
}

func info(con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	infoTask, err := con.Rpc.Info(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleInfo,
	})
	if err != nil {
		console.Log.Errorf("Info error: %v", err)
		return
	}
	con.AddCallback(infoTask.TaskId, func(msg proto.Message) {
		con.SessionLog(session.SessionId).Consolef("Info: %v\n", msg.(*implantpb.Spite).Body)
	})
}
