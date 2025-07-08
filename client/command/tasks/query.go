package tasks

import (
	"fmt"
	"strconv"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
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
	con.GetInteractive().Console(task, fmt.Sprintf("query task %d", id))
	return nil
}

func QueryTask(rpc clientrpc.MaliceRPCClient, session *core.Session, taskId uint32) (*clientpb.Task, error) {
	return rpc.QueryTask(session.Context(), &implantpb.TaskCtrl{
		TaskId: taskId,
		Op:     consts.ModuleQueryTask,
	})
}
