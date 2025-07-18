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
	"strconv"
)

func CancelTaskCmd(cmd *cobra.Command, con *repl.Console) error {
	taskId := cmd.Flags().Arg(0)
	id, err := strconv.Atoi(taskId)
	if err != nil {
		return err
	}

	task, err := CancelTask(con.Rpc, con.GetInteractive(), uint32(id))
	if err != nil {
		return err
	}

	con.GetInteractive().Console(cmd, task, fmt.Sprintf("cancel task %d", id))
	return nil
}

func CancelTask(rpc clientrpc.MaliceRPCClient, session *core.Session, taskId uint32) (*clientpb.Task, error) {
	return rpc.CancelTask(session.Context(), &implantpb.TaskCtrl{
		TaskId: taskId,
		Op:     consts.ModuleCancelTask,
	})
}
