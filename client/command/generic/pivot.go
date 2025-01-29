package generic

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func ListPivotCmd(cmd *cobra.Command, con *repl.Console) error {
	agents, err := ListPivot(con)
	if err != nil {
		return err
	}

	if len(agents) == 0 {
		logs.Log.Info("No pivots\n")
		return nil
	}

	tui.RendStructDefault(agents)
	return nil
}

func ListPivot(con *repl.Console) ([]*clientpb.REMAgent, error) {
	pivots, err := con.GetPivots(con.Context(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	return pivots.Agents, nil
}
