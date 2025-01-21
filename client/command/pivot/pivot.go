package pivot

import (
	"fmt"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ListPivotCmd(cmd *cobra.Command, con *repl.Console) error {
	agents, err := ListPivot(con)
	if err != nil {
		return err
	}

	if len(agents) == 0 {
		logs.Log.Info("No pivots")
		return nil
	}

	// 新增：渲染表格
	var rowEntries []table.Row
	for _, agent := range agents {
		row := table.RowData{
			"ID":     agent.Id,
			"Mod":    agent.Mod,
			"Local":  agent.Local,
			"Remote": agent.Remote,
		}
		rowEntries = append(rowEntries, table.NewRow(row))
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 20),
		table.NewColumn("Mod", "Mod", 20),
		table.NewColumn("Local", "Local", 20),
		table.NewColumn("Remote", "Remote", 20),
	}, true)

	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}

func ListPivot(con *repl.Console) ([]*clientpb.REMAgent, error) {
	pivots, err := con.ListPivots(con.Context(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	return pivots.Agents, nil
}
