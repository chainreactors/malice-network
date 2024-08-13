package sys

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"google.golang.org/protobuf/proto"
	"os"
)

func NetstatCmd(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	sid := con.GetInteractive().SessionId
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
	resultChan := make(chan *implantpb.NetstatResponse)
	con.AddCallback(killTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetNetstatResponse()
		resultChan <- resp
	})
	result := <-resultChan
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "LocalAddr", Width: 15},
		{Title: "RemoteAddr", Width: 15},
		{Title: "SkState", Width: 7},
		{Title: "Pid", Width: 7},
		{Title: "Protocol", Width: 10},
	}, true)
	for _, sock := range result.GetSocks() {
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
	fmt.Printf(tableModel.View(), os.Stdout)
	//newTable := tui.NewModel(tableModel, nil, false, false)
	//err = newTable.Run()
	//if err != nil {
	//	con.SessionLog(sid).Errorf("Error running table: %v", err)
	//}
}
