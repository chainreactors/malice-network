package sys

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func KillCmd(cmd *cobra.Command, con *repl.Console) {
	pid := cmd.Flags().Arg(0)
	if pid == "" {
		repl.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	sid := con.GetInteractive().SessionId
	if session == nil {
		return
	}
	task, err := Kill(con.Rpc, session, pid)
	if err != nil {
		repl.Log.Errorf("Kill error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Killed process\n")
	})
}

func Kill(rpc clientrpc.MaliceRPCClient, session *repl.Session, pid string) (*clientpb.Task, error) {
	task, err := rpc.Kill(repl.Context(session), &implantpb.Request{
		Name:  consts.ModuleKill,
		Input: pid,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
