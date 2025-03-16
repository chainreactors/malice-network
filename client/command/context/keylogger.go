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

func GetKeyloggersCmd(cmd *cobra.Command, con *repl.Console) error {
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
			"Session": ctx.Session.SessionId,
			"Task":    getTaskId(ctx.Task),
			"Name":    keylogger.Name,
			"Path":    keylogger.FilePath,
			"Size":    fmt.Sprintf("%.2f KB", float64(keylogger.Size)/1024),
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 36),
		table.NewColumn("Session", "Session", 16),
		table.NewColumn("Task", "Task", 8),
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Path", "Path", 40),
		table.NewColumn("Size", "Size", 12),
	}, true)

	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}

func GetKeyloggers(con *repl.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextKeyLogger)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddKeylogger(con *repl.Console, sess *core.Session, task *clientpb.Task, data []byte) (bool, error) {
	_, err := con.Rpc.AddKeylogger(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextKeyLogger,
		Value:   output.MarshalContext(&output.KeyLoggerContext{Content: data}),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterKeylogger(con *repl.Console) {
	con.RegisterServerFunc("keyloggers", func(con *repl.Console) ([]*output.KeyLoggerContext, error) {
		keyloggers, err := GetKeyloggers(con)
		if err != nil {
			return nil, err
		}
		return output.ToContexts[*output.KeyLoggerContext](keyloggers)
	}, nil)
	con.RegisterServerFunc("add_keylogger", AddKeylogger, nil)
}
