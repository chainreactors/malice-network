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
			"Session":   getSessionID(ctx.Session),
			"Task":      getTaskId(ctx.Task),
			"Type":      ctx.Type,
			"CreatedAt": ctx.CreatedAt,
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("ID", "ID", 1),
		table.NewColumn("Session", "Session", 10),
		table.NewColumn("Task", "Task", 6),
		table.NewColumn("Type", "Type", 12),
		table.NewColumn("CreatedAt", "Created At", 20),
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

func getSessionID(session *clientpb.Session) string {
	if session == nil {
		return "-"
	}
	return session.SessionId
}

func GetContextsByTask(con *core.Console, contextType string, task *clientpb.Task) (*clientpb.Contexts, error) {
	allContexts, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: contextType,
		Task: task,
	})
	if err != nil {
		return nil, err
	}

	return allContexts, nil
}
