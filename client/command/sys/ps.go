package sys

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
	"strconv"
)

func PsCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := Ps(con.Rpc, session)
	if err != nil {
		repl.Log.Errorf("Ps error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
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
		fmt.Printf(tableModel.View())
	})
}

func Ps(rpc clientrpc.MaliceRPCClient, session *repl.Session) (*clientpb.Task, error) {
	task, err := rpc.Ps(repl.Context(session), &implantpb.Request{
		Name: consts.ModulePs,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
