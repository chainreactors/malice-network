package service

import (
	"errors"
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
)

func ServiceListCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := ServiceList(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, "service list")
	return nil
}

func ServiceList(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	return rpc.ServiceList(session.Context(), &implantpb.Request{
		Name: consts.ModuleServiceList,
	})
}

func RegisterServiceListFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleServiceList,
		ServiceList,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			return fmt.Sprintf("%v", content.Spite.GetBody()), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			services := content.Spite.GetServicesResponse().GetServices()
			if len(services) == 0 {
				return "", errors.New("no services found")
			}

			tableModel := tui.NewTable([]table.Column{
				table.NewColumn("Name", "Name", 20),
				table.NewColumn("Display Name", "Display Name", 25),
				table.NewColumn("Executable Path", "Executable Path", 40),
				table.NewColumn("Start Type", "Start Type", 10),
				table.NewColumn("Error Control", "Error Control", 10),
				table.NewColumn("Account Name", "Account Name", 20),
				table.NewColumn("Current State", "Current State", 15),
				table.NewColumn("Process ID", "Process ID", 10),
				table.NewColumn("Exit Code", "Exit Code", 10),
				table.NewColumn("Checkpoint", "Checkpoint", 12),
				table.NewColumn("Wait Hint", "Wait Hint", 12),
				//{Title: "Name", Width: 20},
				//{Title: "Display Name", Width: 25},
				//{Title: "Executable Path", Width: 60},
				//{Title: "Start Type", Width: 10},
				//{Title: "Error Control", Width: 10},
				//{Title: "Account Name", Width: 20},
				//{Title: "Current State", Width: 10},
				//{Title: "Process ID", Width: 10},
				//{Title: "Exit Code", Width: 10},
				//{Title: "Checkpoint", Width: 12},
				//{Title: "Wait Hint", Width: 12},
			}, true)

			var rowEntries []table.Row
			for _, service := range services {
				row := table.NewRow(
					table.RowData{
						"Name":            service.Config.Name,
						"Display Name":    service.Config.DisplayName,
						"Executable Path": service.Config.ExecutablePath,
						"Start Type":      strconv.Itoa(int(service.Config.StartType)),
						"Error Control":   strconv.Itoa(int(service.Config.ErrorControl)),
						"Account Name":    service.Config.AccountName,
						"Current State":   strconv.Itoa(int(service.Status.CurrentState)),
						"Process ID":      strconv.Itoa(int(service.Status.ProcessId)),
						"Exit Code":       strconv.Itoa(int(service.Status.ExitCode)),
						"Checkpoint":      strconv.Itoa(int(service.Status.Checkpoint)),
						"Wait Hint":       strconv.Itoa(int(service.Status.WaitHint)),
					})
				//	table.Row{
				//	service.Config.Name,
				//	service.Config.DisplayName,
				//	service.Config.ExecutablePath,
				//	strconv.Itoa(int(service.Config.StartType)),
				//	strconv.Itoa(int(service.Config.ErrorControl)),
				//	service.Config.AccountName,
				//	strconv.Itoa(int(service.Status.CurrentState)),
				//	strconv.Itoa(int(service.Status.ProcessId)),
				//	strconv.Itoa(int(service.Status.ExitCode)),
				//	strconv.Itoa(int(service.Status.Checkpoint)),
				//	strconv.Itoa(int(service.Status.WaitHint)),
				//}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetMultiline()
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		},
	)
	con.AddCommandFuncHelper(
		consts.ModuleServiceList,
		consts.ModuleServiceList,
		consts.ModuleServiceList+"(active())",
		[]string{
			"session: special session",
		},
		[]string{"task"})
}
