package context

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ListContexts(cmd *cobra.Command, con *core.Console) error {
	contexts, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{})
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range contexts.Contexts {
		row := table.NewRow(table.RowData{
			"ID":        ctx.Id,
			"Session":   ctx.Session.SessionId,
			"Task":      getTaskId(ctx.Task),
			"Type":      ctx.Type,
			"CreatedAt": ctx.CreatedAt,
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 36),
		table.NewColumn("Session", "Session", 8),
		table.NewColumn("Task", "Task", 8),
		table.NewColumn("Type", "Type", 12),
		table.NewColumn("CreatedAt", "CreatedAt", 20),
	}, true)

	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}

func GetContextsByType(con *core.Console, contextType string) (*clientpb.Contexts, error) {
	allContexts, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: contextType,
	})
	if err != nil {
		return nil, err
	}

	return allContexts, nil
}

func GetContextsByTask(con *core.Console, contextType string, task *clientpb.Task) (*clientpb.Contexts, error) {
	allContexts, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Task: task,
	})
	if err != nil {
		return nil, err
	}

	return allContexts, nil
}
