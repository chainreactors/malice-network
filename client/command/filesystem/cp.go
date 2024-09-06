package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
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

	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := Cp(con.Rpc, con.GetInteractive(), originPath, targetPath)
	if err != nil {
		console.Log.Errorf("Cp error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		_ = msg.(*clientpb.Task)
		con.SessionLog(sid).Consolef("Cp success\n")
	})
}

func Cp(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, originPath, targetPath string) (*clientpb.Task, error) {
	task, err := rpc.Cp(console.Context(session), &implantpb.Request{
		Name: consts.ModuleCp,
		Args: []string{originPath, targetPath},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
