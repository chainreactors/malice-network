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

func GetKeyloggersCmd(cmd *cobra.Command, con *core.Console) error {
	keyloggers, err := GetKeyloggers(con)
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range keyloggers {
		keylogger, err := output.ToContext[*output.KeyLoggerContext](ctx)
		if err != nil {
			return err
		}

		row := table.NewRow(table.RowData{
			"ID":      ctx.Id,
			"Session": getSessionID(ctx.Session),
			"Task":    getTaskId(ctx.Task),
			"Name":    keylogger.Name,
			"Path":    keylogger.FilePath,
			"Size":    fmt.Sprintf("%.2f KB", float64(keylogger.Size)/1024),
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("ID", "ID", 1),
		table.NewColumn("Session", "Session", 10),
		table.NewColumn("Task", "Task", 6),
		table.NewColumn("Name", "Name", 20),
		table.NewFlexColumn("Path", "Path", 2),
		table.NewColumn("Size", "Size", 12),
	}, true)

	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}

func GetKeyloggers(con *core.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextKeyLogger)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddKeylogger(con *core.Console, sess *client.Session, task *clientpb.Task, data []byte) (bool, error) {
	_, err := con.Rpc.AddKeylogger(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextKeyLogger,
		Value:   data,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterKeylogger(con *core.Console) {
	con.RegisterServerFunc("keyloggers", func(con *core.Console) ([]*output.KeyLoggerContext, error) {
		keyloggers, err := GetKeyloggers(con)
		if err != nil {
			return nil, err
		}
		return output.ToContexts[*output.KeyLoggerContext](keyloggers)
	}, nil)
	con.RegisterServerFunc("add_keylogger", AddKeylogger, nil)
}
