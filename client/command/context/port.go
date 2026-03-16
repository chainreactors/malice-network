package context

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func GetPortsCmd(cmd *cobra.Command, con *core.Console) error {
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
			"Session": getSessionID(ctx.Session),
			"Task":    getTaskId(ctx.Task),
			"Length":  len(portCtx.Ports),
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("ID", "ID", 1),
		table.NewColumn("Session", "Session", 10),
		table.NewColumn("Task", "Task", 6),
		table.NewColumn("Count", "Count", 8),
	}, true)

	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}

func getTaskId(task *clientpb.Task) string {
	if task == nil {
		return "-"
	}
	return fmt.Sprint(task.TaskId)
}

func GetPorts(con *core.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextPort)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddPort(con *core.Console, sess *client.Session, task *clientpb.Task, ports []*output.Port) (bool, error) {
	if err := requireContextTask(sess, task); err != nil {
		return false, err
	}

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

func RegisterPort(con *core.Console) {
	con.RegisterServerFunc("ports", func(con *core.Console) ([]*output.PortContext, error) {
		ports, err := GetPorts(con)
		if err != nil {
			return nil, err
		}
		return output.ToContexts[*output.PortContext](ports)
	}, nil)
	con.RegisterServerFunc("add_port", AddPort, nil)
}
