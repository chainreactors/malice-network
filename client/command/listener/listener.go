package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"strconv"
)

func ListenerCmd(cmd *cobra.Command, con *repl.Console) error {
	listeners, err := con.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	printListeners(listeners)
	return nil
}

func printListeners(listeners *clientpb.Listeners) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 10),
		table.NewColumn("Addr", "Addr", 15),
		table.NewColumn("Active", "Active", 7),
	}, true)
	for _, listener := range listeners.GetListeners() {
		row = table.NewRow(
			table.RowData{
				"ID":     listener.Id,
				"Addr":   listener.Addr,
				"Active": strconv.FormatBool(listener.Active),
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	tableModel.Title = "listeners"
	fmt.Printf(tableModel.View())
}
