package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"strconv"
)

func ListenerCmd(cmd *cobra.Command, con *repl.Console) {
	listeners, err := con.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		con.Log.Errorf("Failed to list listeners: %s", err)
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
	fmt.Printf(tableModel.View())
}
