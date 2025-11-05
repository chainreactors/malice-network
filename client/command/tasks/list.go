package tasks

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func ListTaskCmd(cmd *cobra.Command, con *repl.Console) error {
	task, err := ListTask(con.Rpc, con.GetInteractive())
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func ListTask(rpc clientrpc.MaliceRPCClient, session *client.Session) (*clientpb.Task, error) {
	return rpc.ListTasks(session.Context(), &implantpb.Request{
		Name: consts.ModuleListTask,
	})
}
