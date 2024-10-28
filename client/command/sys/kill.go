package sys

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func KillCmd(cmd *cobra.Command, con *repl.Console) error {
	pid := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := Kill(con.Rpc, session, pid)
	if err != nil {

		return err
	}
	session.Console(task, "kill "+pid)
	return nil
}

func Kill(rpc clientrpc.MaliceRPCClient, session *core.Session, pid string) (*clientpb.Task, error) {
	task, err := rpc.Kill(session.Context(), &implantpb.Request{
		Name:  consts.ModuleKill,
		Input: pid,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterKillFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleKill,
		Kill,
		"bkill",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, pid string) (*clientpb.Task, error) {
			return Kill(rpc, sess, pid)
		},
		common.ParseStatus,
		nil)
}
