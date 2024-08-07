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
	"strconv"
)

func PsCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	sid := con.ActiveTarget.GetInteractive().SessionId
	if session == nil {
		return
	}
	psTask, err := con.Rpc.Ps(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModulePs,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Ps error: %v", err)
		return
	}
	resultChan := make(chan *implantpb.PsResponse)
	con.AddCallback(psTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetPsResponse()
		resultChan <- resp
	})
	result := <-resultChan
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "PID", Width: 5},
		{Title: "PPID", Width: 5},
		{Title: "Arch", Width: 7},
		{Title: "Owner", Width: 7},
		{Title: "Path", Width: 15},
		{Title: "Args", Width: 10},
	}, true)
	for _, process := range result.GetProcesses() {
		row = table.Row{
			process.Name,
			strconv.Itoa(int(process.Pid)),
			strconv.Itoa(int(process.Ppid)),
			process.Arch,
			process.Owner,
			process.Path,
			process.Args,
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
