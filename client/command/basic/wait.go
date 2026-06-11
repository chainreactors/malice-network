package basic

import (
	"fmt"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/spf13/cobra"
	"strconv"
)

func WaitCmd(cmd *cobra.Command, con *core.Console) error {
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
	if content == nil || content.Task == nil {
		return fmt.Errorf("task %d returned empty response", uintID)
	}
	fn, ok := intermediate.InternalFunctions[content.Task.Type]
	if !ok {
		con.Log.Debugf("function %s not found\n", content.Task.Type)
		return nil
	}

	if fn.FinishCallback != nil {
		data, err := fn.FinishCallback(content)
		if err != nil {
			return err
		}
		session.Log.Console(data)
	}
	return nil
}
