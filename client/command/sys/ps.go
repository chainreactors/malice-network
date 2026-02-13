package sys

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func PsCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	task, err := Ps(con.Rpc, session)
	if err != nil {
		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Ps(rpc clientrpc.MaliceRPCClient, session *client.Session) (*clientpb.Task, error) {
	task, err := rpc.Ps(session.Context(), &implantpb.Request{
		Name: consts.ModulePs,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterPsFunc(con *core.Console) {
	con.RegisterImplantFunc(
		consts.ModulePs,
		Ps,
		"bps",
		func(rpc clientrpc.MaliceRPCClient, sess *client.Session) (*clientpb.Task, error) {
			return Ps(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			psSet := ctx.Spite.GetPsResponse()
			var ps []string
			for _, p := range psSet.GetProcesses() {
				ps = append(ps, fmt.Sprintf("%s:%d:%d:%s:%s:%s:%s",
					p.Name,
					p.Pid,
					p.Ppid,
					p.Arch,
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
				table.NewColumn("Name", "Name", 15),
				table.NewColumn("PID", "PID", 6),
				table.NewColumn("PPID", "PPID", 6),
				table.NewColumn("Arch", "Arch", 6),
				table.NewColumn("Owner", "Owner", 20),
				table.NewFlexColumn("Path", "Path", 2),
				table.NewFlexColumn("Args", "Args", 2),
			}, true)
			for _, process := range resp.GetProcesses() {
				row = table.NewRow(
					table.RowData{
						"Name":  process.Name,
						"PID":   strconv.Itoa(int(process.Pid)),
						"PPID":  strconv.Itoa(int(process.Ppid)),
						"Arch":  process.Arch,
						"UID":   process.Uid,
						"Owner": process.Owner,
						"Path":  process.Path,
						"Args":  process.Args,
					})
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetMultiline()
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.AddCommandFuncHelper(
		consts.ModulePs,
		consts.ModulePs,
		"ps(active)",
		[]string{
			"sess:special session",
		},
		[]string{"task"})
}
