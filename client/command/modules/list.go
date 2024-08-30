package modules

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ListModulesCmd(cmd *cobra.Command, con *console.Console) {
	listModules(con)
}

func listModules(con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	listTask, err := con.Rpc.ListModules(con.ActiveTarget.Context(), &implantpb.Request{Name: consts.ModuleListModule})
	if err != nil {
		console.Log.Errorf("ListModules error: %v", err)
		return
	}
	con.AddCallback(listTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetModules()
		var rowEntries []table.Row
		var row table.Row
		tableModel := tui.NewTable([]table.Column{
			{Title: "Name", Width: 15},
			{Title: "Help", Width: 30},
		}, true)
		for _, module := range resp.GetModules() {
			row = table.Row{
				module,
				"",
			}
			rowEntries = append(rowEntries, row)
		}
		tableModel.SetRows(rowEntries)
		fmt.Printf(tableModel.View())
	})
}
