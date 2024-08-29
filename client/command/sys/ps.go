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
	"os"
	"strconv"
)

func PsCmd(cmd *cobra.Command, con *console.Console) {
	ps(con)
}

func ps(con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	psTask, err := con.Rpc.Ps(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModulePs,
	})
	if err != nil {
		console.Log.Errorf("Ps error: %v", err)
		return
	}
	con.AddCallback(psTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetPsResponse()
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
		for _, process := range resp.GetProcesses() {
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
	})
}
