package modules

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ListModulesCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	task, err := ListModules(con.Rpc, session)
	if err != nil {
		repl.Log.Errorf("ListModules error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
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

func ListModules(rpc clientrpc.MaliceRPCClient, session *repl.Session) (*clientpb.Task, error) {
	listTask, err := rpc.ListModule(repl.Context(session), &implantpb.Request{Name: consts.ModuleListModule})
	if err != nil {
		return nil, err
	}
	return listTask, nil
}
