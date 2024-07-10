package listener

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"strconv"
)

func ListenerCmd(ctx *grumble.Context, con *console.Console) {
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
		{Title: "Pl_Name", Width: 10},
		{Title: "Pl_Host", Width: 10},
		{Title: "Pl_Port", Width: 10},
		{Title: "Pl_Type", Width: 10},
	}, true)
	for _, listener := range listeners.GetListeners() {
		for _, pipeline := range listener.Pipelines.Pipelines {
			row = table.Row{listener.Id,
				listener.Addr,
				strconv.FormatBool(listener.Active),
				pipeline.GetTcp().Name,
				pipeline.GetTcp().Host,
				strconv.Itoa(int(pipeline.GetTcp().Port)),
				"tcp",
			}
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	newTable := tui.NewModel(tableModel, nil, false, false)
	err := newTable.Run()
	if err != nil {
		console.Log.Errorf("Failed to run table: %s", err)
		return
	}
}
