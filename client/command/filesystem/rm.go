package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func RmCmd(cmd *cobra.Command, con *console.Console) {
	fileName := cmd.Flags().Arg(0)
	if fileName == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	rm(fileName, con)
}

func rm(fileName string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	rmTask, err := con.Rpc.Rm(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleRm,
		Input: fileName,
	})
	if err != nil {
		console.Log.Errorf("Rm error: %v", err)
		return
	}
	con.AddCallback(rmTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Removed file success\n")
	})
}
