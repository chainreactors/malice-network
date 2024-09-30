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

func CancelTaskCmd(cmd *cobra.Command, con *repl.Console) {
	taskId := cmd.Flags().Arg(0)
	id, err := strconv.Atoi(taskId)
	if err != nil {
		con.Log.Errorf("Error converting taskId to int: %s", err)
		return
	}

	task, err := CancelTask(con.Rpc, con.GetInteractive(), uint32(id))
	if err != nil {
		con.Log.Errorf("Error canceling task: %s", err)
		return
	}

	con.GetInteractive().Console(task, fmt.Sprintf("cancel task %d", id))
}

func CancelTask(rpc clientrpc.MaliceRPCClient, session *core.Session, taskId uint32) (*clientpb.Task, error) {
	return rpc.CancelTask(session.Context(), &implantpb.ImplantTask{
		TaskId: taskId,
		Op:     consts.ModuleCancelTask,
	})
}
