package sys

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func KillCmd(cmd *cobra.Command, con *console.Console) {
	pid := cmd.Flags().Arg(0)
	if pid == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	sid := con.GetInteractive().SessionId
	if session == nil {
		return
	}
	task, err := Kill(con.Rpc, session, pid)
	if err != nil {
		console.Log.Errorf("Kill error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Killed process\n")
	})
}

func Kill(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, pid string) (*clientpb.Task, error) {
	task, err := rpc.Kill(console.Context(session), &implantpb.Request{
		Name:  consts.ModuleKill,
		Input: pid,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
