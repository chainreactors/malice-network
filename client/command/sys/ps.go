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
	"github.com/evertras/bubble-table/table"
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
				table.NewColumn("Name", "Name", 20),
				table.NewColumn("PID", "PID", 5),
				table.NewColumn("PPID", "PPID", 5),
				table.NewColumn("Arch", "Arch", 7),
				table.NewColumn("UID", "UID", 36),
				table.NewColumn("Owner", "Owner", 7),
				table.NewColumn("Path", "Path", 30),
				table.NewColumn("Args", "Args", 30),
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
