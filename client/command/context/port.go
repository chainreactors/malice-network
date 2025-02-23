package context

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func GetPortsCmd(cmd *cobra.Command, con *repl.Console) error {
	ports, err := GetPorts(con)
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range ports {
		portCtx, err := output.ToContext[*output.PortContext](ctx)
		if err != nil {
			return err
		}

		row := table.NewRow(table.RowData{
			"ID":      ctx.Id,
			"Session": ctx.Session.SessionId,
			"Task":    getTaskId(ctx.Task),
			"Length":  len(portCtx.Ports),
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 36),
		table.NewColumn("Session", "Session", 8),
		table.NewColumn("Task", "Task", 8),
		table.NewColumn("Count", "Count", 8),
	}, true)

	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}

func getTaskId(task *clientpb.Task) string {
	if task == nil {
		return "-"
	}
	return fmt.Sprint(task.TaskId)
}

func GetPorts(con *repl.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextPort)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddPort(con *repl.Console, sess *core.Session, task *clientpb.Task, ports []*output.Port) (bool, error) {
	_, err := con.Rpc.AddPort(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextPort,
		Value:   output.MarshalContext(&output.PortContext{Ports: ports}),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterPort(con *repl.Console) {
	con.RegisterServerFunc("ports", GetPorts, nil)
	con.RegisterServerFunc("add_port", AddPort, nil)
}
