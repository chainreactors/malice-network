package listener

import (
	"context"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"strconv"
)

func ListenerCmd(cmd *cobra.Command, con *console.Console) {
	listeners, err := con.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		console.Log.Errorf("Failed to list listeners: %s", err)
		return
	}
	printListeners(listeners)
}

func printListeners(listeners *clientpb.Listeners) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "ID", Width: 10},
		{Title: "Addr", Width: 15},
		{Title: "Active", Width: 7},
	}, true)
	for _, listener := range listeners.GetListeners() {
		row = table.Row{listener.Id,
			listener.Addr,
			strconv.FormatBool(listener.Active),
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	tableModel.Title = "listeners"
	newTable := tui.NewModel(tableModel, nil, false, false)
	err := newTable.Run()
	if err != nil {
		console.Log.Errorf("Failed to run table: %s", err)
		return
	}
}
