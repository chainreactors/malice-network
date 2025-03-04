package generic

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func ListPivotCmd(cmd *cobra.Command, con *repl.Console) error {
	all, _ := cmd.Flags().GetBool("all")
	agents, err := ListPivot(con)
	if err != nil {
		return err
	}

	if len(agents) == 0 {
		logs.Log.Info("No pivots\n")
		return nil
	}

	for _, c := range agents {
		if all {
			logs.Log.Info(c.String() + "\n")
		} else if c.Enable {
			logs.Log.Info(c.String() + "\n")
		}

	}
	return nil
}

func ListPivot(con *repl.Console) ([]*output.PivotingContext, error) {
	pivots, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: consts.ContextPivoting,
	})
	if err != nil {
		return nil, err
	}
	ctxs, err := output.ToContexts[*output.PivotingContext](pivots.Contexts)
	return ctxs, nil
}
