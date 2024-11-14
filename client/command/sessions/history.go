package sessions

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"strconv"
)

func historyCmd(cmd *cobra.Command, con *repl.Console) error {
	if con.GetInteractive() == nil {
		con.Log.Errorf("No session selected")
		return nil
	}

	rawLen := cmd.Flags().Arg(0)
	if rawLen == "" {
		rawLen = "10"
	}
	length, err := strconv.Atoi(rawLen)
	if err != nil {
		return err
	}
	contexts, err := con.Rpc.GetSessionHistory(con.GetInteractive().Context(), &clientpb.SessionLog{
		SessionId: con.GetInteractive().SessionId,
		Limit:     int32(length),
	})
	if err != nil {
		return err
	}
	log := con.ServerStatus.ObserverLog(con.GetInteractive().SessionId)
	for _, context := range contexts.Contexts {
		if fn, ok := intermediate.InternalFunctions[context.Task.Type]; ok && fn.FinishCallback != nil {
			err := core.HandleTaskContext(log, &clientpb.TaskContext{
				Task:    context.Task,
				Session: context.Session,
				Spite:   context.Spite,
			}, fn, false, "")
			if err != nil {
				return err
			}
		} else {
			log.Consolef("%s not impl output impl\n", context.Task.Type)
		}
	}
	return nil
}
