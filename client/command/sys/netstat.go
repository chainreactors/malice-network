package sys

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

func NetstatCmd(cmd *cobra.Command, con *console.Console) {
	netstat(con)
}

func netstat(con *console.Console) {
	netstatTask, err := con.Rpc.Netstat(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleNetstat,
	})
	if err != nil {
		console.Log.Errorf("Kill error: %v", err)
		return
	}
	con.AddCallback(netstatTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetNetstatResponse()
		var rowEntries []table.Row
		var row table.Row
		tableModel := tui.NewTable([]table.Column{
			{Title: "LocalAddr", Width: 15},
			{Title: "RemoteAddr", Width: 15},
			{Title: "SkState", Width: 7},
			{Title: "Pid", Width: 7},
			{Title: "Protocol", Width: 10},
		}, true)
		for _, sock := range resp.GetSocks() {
			row = table.Row{
				sock.LocalAddr,
				sock.RemoteAddr,
				sock.SkState,
				sock.Pid,
				sock.Protocol,
			}
			rowEntries = append(rowEntries, row)
		}
		tableModel.SetRows(rowEntries)
		fmt.Printf(tableModel.View())
	})
}
