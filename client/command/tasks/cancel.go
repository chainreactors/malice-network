package tasks

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"gorm.io/gorm/utils"
	"strconv"
)

func CancelTaskCmd(cmd *cobra.Command, con *repl.Console) {
	taskId := cmd.Flags().Arg(0)
	id, err := strconv.Atoi(taskId)
	if err != nil {
		con.Log.Errorf("Error converting taskId to int: %s", err)
		return
	}
	_, err = CancelTask(con.Rpc, con.GetInteractive(), uint32(id))
	if err != nil {
		con.Log.Errorf("Error canceling task: %s", err)
		return
	}
}

func CancelTask(rpc clientrpc.MaliceRPCClient, session *repl.Session, taskId uint32) (*clientpb.Task, error) {
	if session.HasTask(taskId) {
		return nil, fmt.Errorf("task %d not found in %s", taskId, session.SessionId)
	}

	return rpc.CancelTask(session.Context(), &implantpb.Request{Input: utils.ToString(taskId)})
}
