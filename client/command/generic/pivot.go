package generic

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
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

	PrintPivots(agents, con, all)
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

func PrintPivots(pivots []*output.PivotingContext, con *repl.Console, all bool) {
	var rowEntries []table.Row
	for _, pivot := range pivots {
		row := table.NewRow(
			table.RowData{
				"Enable":     fmt.Sprintf("%t", pivot.Enable),
				"Listener":   pivot.Listener,
				"Pipeline":   pivot.Pipeline,
				"RemAgentID": pivot.RemAgentID,
				"LocalURL":   pivot.LocalURL,
				"RemoteURL":  pivot.RemoteURL,
				"Mod":        pivot.Mod,
			})
		if all || pivot.Enable {
			rowEntries = append(rowEntries, row)
		}
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Enable", "Enable", 6),
		table.NewColumn("Listener", "Listener", 10),
		table.NewColumn("Pipeline", "Pipeline", 10),
		table.NewColumn("RemAgentID", "RemAgentID", 10),
		table.NewColumn("LocalURL", "LocalURL", 50),
		table.NewColumn("RemoteURL", "RemoteURL", 50),
		table.NewColumn("Mod", "Mod", 10),
	}, true)

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	//tui.Reset()
}
