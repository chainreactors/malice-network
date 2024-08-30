package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func CpCmd(cmd *cobra.Command, con *console.Console) {
	originPath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)
	if originPath == "" || targetPath == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	args := []string{originPath, targetPath}
	cp(args, con)
}

func cp(args []string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	cpTask, err := con.Rpc.Cp(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleCp,
		Args: args,
	})
	if err != nil {
		console.Log.Errorf("Cp error: %v", err)
		return
	}
	con.AddCallback(cpTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Cp success\n")
	})
}
