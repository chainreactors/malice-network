package sys

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
)

func PsCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := Ps(con.Rpc, session)
	if err != nil {
		return err
	}
	session.Console(task, "ps")
	return nil
}

func Ps(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Ps(session.Context(), &implantpb.Request{
		Name: consts.ModulePs,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterPsFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModulePs,
		Ps,
		"bps",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
			return Ps(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			psSet := ctx.Spite.GetPsResponse()
			var ps []string
			for _, p := range psSet.GetProcesses() {
				ps = append(ps, fmt.Sprintf("%s:%d:%d:%s:%s:%s:%s:%s",
					p.Name,
					p.Pid,
					p.Ppid,
					p.Arch,
					p.Uid,
					p.Owner,
					p.Path,
					p.Args))
			}
			return strings.Join(ps, ","), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			resp := content.Spite.GetPsResponse()
			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				{Title: "Name", Width: 20},
				{Title: "PID", Width: 5},
				{Title: "PPID", Width: 5},
				{Title: "Arch", Width: 7},
				{Title: "UID", Width: 36},
				{Title: "Owner", Width: 7},
				{Title: "Path", Width: 50},
				{Title: "Args", Width: 50},
			}, true)
			for _, process := range resp.GetProcesses() {
				row = table.Row{
					process.Name,
					strconv.Itoa(int(process.Pid)),
					strconv.Itoa(int(process.Ppid)),
					process.Arch,
					process.Uid,
					process.Owner,
					process.Path,
					process.Args,
				}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.AddInternalFuncHelper(
		consts.ModulePs,
		consts.ModulePs,
		"ps(active)",
		[]string{
			"sess:special session",
		},
		[]string{"task"})
}
