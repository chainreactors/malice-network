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
	"strings"
)

func NetstatCmd(cmd *cobra.Command, con *repl.Console) error {
	task, err := Netstat(con.Rpc, con.GetInteractive())
	if err != nil {
		return err
	}
	con.GetInteractive().Console(cmd, task, "netstat")
	return nil
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

func RegisterNetstatFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleNetstat,
		Netstat,
		"bnetstat",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
			return Netstat(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			netstatSet := ctx.Spite.GetNetstatResponse()
			var socks []string
			for _, sock := range netstatSet.GetSocks() {
				socks = append(socks, fmt.Sprintf("%s:%s:%s:%s:%s",
					sock.LocalAddr,
					sock.RemoteAddr,
					sock.SkState,
					sock.Pid,
					sock.Protocol))
			}
			return strings.Join(socks, ","), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			resp := content.Spite.GetNetstatResponse()
			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				table.NewColumn("LocalAddr", "LocalAddr", 30),
				table.NewColumn("RemoteAddr", "RemoteAddr", 30),
				table.NewColumn("SkState", "SkState", 20),
				table.NewColumn("Pid", "Pid", 7),
				table.NewColumn("Protocol", "Protocol", 10),
			}, true)
			for _, sock := range resp.GetSocks() {
				row = table.NewRow(
					table.RowData{
						"LocalAddr":  sock.LocalAddr,
						"RemoteAddr": sock.RemoteAddr,
						"SkState":    sock.SkState,
						"Pid":        sock.Pid,
						"Protocol":   sock.Protocol,
					})
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetMultiline()
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.AddCommandFuncHelper(
		consts.ModuleNetstat,
		consts.ModuleNetstat,
		"netstat(active)",
		[]string{
			"sess: special session",
		},
		[]string{"task"})
}
