package context

import (
	"fmt"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func GetScreenshotsCmd(cmd *cobra.Command, con *repl.Console) error {
	screenshots, err := GetScreenshots(con)
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range screenshots {
		screenshot, err := types.ToContext[*types.ScreenShotContext](ctx)
		if err != nil {
			return err
		}
		row := table.NewRow(table.RowData{
			"ID":      ctx,
			"Session": ctx.Session.SessionId,
			"Task":    getTaskId(ctx.Task),
			"Name":    screenshot.Name,
			"Path":    screenshot.FilePath,
			"Size":    fmt.Sprintf("%.2f KB", float64(screenshot.Size)/1024),
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

func GetScreenshots(con *repl.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextScreenShot)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddScreenshot(con *repl.Console, sess *core.Session, task *clientpb.Task, data []byte) (bool, error) {
	_, err := con.Rpc.AddScreenShot(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextScreenShot,
		Value: types.MarshalContext(&types.ScreenShotContext{
			Content: data,
		}),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterScreenshot(con *repl.Console) {
	con.RegisterServerFunc("screenshots", GetScreenshots, nil)
	con.RegisterServerFunc("add_screenshot", AddScreenshot, nil)
}
