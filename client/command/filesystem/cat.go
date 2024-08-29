package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func CatCmd(cmd *cobra.Command, con *console.Console) {
	fileName := cmd.Flags().Arg(0)
	if fileName == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	cat(fileName, con)
}

func cat(fileName string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	catTask, err := con.Rpc.Cat(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleCat,
		Input: fileName,
	})
	if err != nil {
		console.Log.Errorf("Cat error: %v", err)
		return
	}
	con.AddCallback(catTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		con.SessionLog(sid).Consolef("File content: %s\n", resp.GetOutput())
	})
}
