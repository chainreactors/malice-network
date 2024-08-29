package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func MvCmd(cmd *cobra.Command, con *console.Console) {
	sourcePath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)
	if sourcePath == "" || targetPath == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	args := []string{sourcePath, targetPath}
	mv(args, con)
}

func mv(args []string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	mvTask, err := con.Rpc.Mv(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleMv,
		Args: args,
	})
	if err != nil {
		console.Log.Errorf("Mv error: %v", err)
		return
	}
	con.AddCallback(mvTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Mv success\n")
	})
}
