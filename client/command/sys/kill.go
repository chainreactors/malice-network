package sys

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func KillCmd(cmd *cobra.Command, con *core.Console) error {
	pid := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := Kill(con.Rpc, session, pid)
	if err != nil {

		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Kill(rpc clientrpc.MaliceRPCClient, session *client.Session, pid string) (*clientpb.Task, error) {
	task, err := rpc.Kill(session.Context(), &implantpb.Request{
		Name:  consts.ModuleKill,
		Input: pid,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterKillFunc(con *core.Console) {
	con.RegisterImplantFunc(
		consts.ModuleKill,
		Kill,
		"bkill",
		func(rpc clientrpc.MaliceRPCClient, sess *client.Session, pid string) (*clientpb.Task, error) {
			return Kill(rpc, sess, pid)
		},
		output.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleKill,
		consts.ModuleKill,
		consts.ModuleKill+"(active(),pid)",
		[]string{
			"session: special session",
			"pid: process id",
		},
		[]string{"task"})

}
