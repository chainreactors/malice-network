package tasks

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func ListTaskCmd(cmd *cobra.Command, con *repl.Console) error {
	task, err := ListTask(con.Rpc, con.GetInteractive())
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, fmt.Sprintf("list_task"))
	return nil
}

func ListTask(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	return rpc.ListTasks(session.Context(), &implantpb.Request{
		Name: consts.ModuleListTask,
	})
}
