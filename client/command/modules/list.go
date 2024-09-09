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
	task, err := ListModules(con.Rpc, session)
	if err != nil {
		repl.Log.Errorf("ListModules error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		modules := msg.(*implantpb.Spite).GetModules()
		if len(modules.Modules) == 0 {
			session.Log.Warn("No modules found.")
			return
		}

		var rowEntries []table.Row
		var row table.Row
		tableModel := tui.NewTable([]table.Column{
			{Title: "Name", Width: 15},
			{Title: "Help", Width: 30},
		}, true)
		for _, module := range modules.GetModules() {
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
	listTask, err := rpc.ListModule(session.Context(), &implantpb.Request{Name: consts.ModuleListModule})
	if err != nil {
		return nil, err
	}
	return listTask, nil
}
