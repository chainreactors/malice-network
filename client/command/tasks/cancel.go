package tasks

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
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
	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func CancelTask(rpc clientrpc.MaliceRPCClient, session *client.Session, taskId uint32) (*clientpb.Task, error) {
	return rpc.CancelTask(session.Context(), &implantpb.TaskCtrl{
		TaskId: taskId,
		Op:     consts.ModuleCancelTask,
	})
}
