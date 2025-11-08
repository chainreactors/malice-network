package tasks

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"strconv"

	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func QueryTaskCmd(cmd *cobra.Command, con *repl.Console) error {
	taskId := cmd.Flags().Arg(0)
	id, err := strconv.Atoi(taskId)
	if err != nil {
		return err
	}

	task, err := QueryTask(con.Rpc, con.GetInteractive(), uint32(id))
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func QueryTask(rpc clientrpc.MaliceRPCClient, session *client.Session, taskId uint32) (*clientpb.Task, error) {
	return rpc.QueryTask(session.Context(), &implantpb.TaskCtrl{
		TaskId: taskId,
		Op:     consts.ModuleQueryTask,
	})
}
