package basic

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"strconv"
)

func WaitCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	taskID := cmd.Flags().Arg(0)
	uintID, err := strconv.Atoi(taskID)
	if err != nil {
		return err
	}
	content, err := con.Rpc.WaitTaskFinish(session.Context(), &clientpb.Task{
		TaskId:    uint32(uintID),
		SessionId: session.SessionId,
	})
	if err != nil {
		return err
	}
	con.Log.Info(content.Spite)
	return nil
}
