package sys

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
)

func NetstatCmd(cmd *cobra.Command, con *repl.Console) {
	task, err := Netstat(con.Rpc, con.GetInteractive())
	if err != nil {
		con.Log.Errorf("Kill error: %v", err)
		return
	}
	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		resp := msg.GetNetstatResponse()
		var rowEntries []table.Row
		var row table.Row
		tableModel := tui.NewTable([]table.Column{
			{Title: "LocalAddr", Width: 30},
			{Title: "RemoteAddr", Width: 30},
			{Title: "SkState", Width: 20},
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
		return tableModel.View(), nil
	})
}

func Netstat(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Netstat(session.Context(), &implantpb.Request{
		Name: consts.ModuleNetstat,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
