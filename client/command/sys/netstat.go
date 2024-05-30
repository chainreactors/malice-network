package sys

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/charmbracelet/bubbles/table"
	"google.golang.org/protobuf/proto"
)

func NetstatCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	sid := con.ActiveTarget.GetInteractive().SessionId
	if session == nil {
		return
	}
	killTask, err := con.Rpc.Netstat(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleNetstat,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Kill error: %v", err)
		return
	}
	con.AddCallback(killTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetNetstatResponse()
		var rowEntries []table.Row
		var row table.Row
		tableModel := tui.NewTable([]table.Column{
			{Title: "LocalAddr", Width: 15},
			{Title: "RemoteAddr", Width: 15},
			{Title: "SkState", Width: 7},
			{Title: "Pid", Width: 7},
			{Title: "Protocol", Width: 10},
		})
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
		tableModel.Rows = rowEntries
		tableModel.SetRows()
		tableModel.SetHandle(func() {})
		err := tui.Run(tableModel)
		if err != nil {
			con.SessionLog(sid).Errorf("Error running table: %v", err)
		}
	})
}
