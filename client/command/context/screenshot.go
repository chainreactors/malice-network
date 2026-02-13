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

func GetScreenshotsCmd(cmd *cobra.Command, con *core.Console) error {
	screenshots, err := GetScreenshots(con)
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range screenshots {
		screenshot, err := output.ToContext[*output.ScreenShotContext](ctx)
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

func GetScreenshots(con *core.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextScreenShot)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddScreenshot(con *core.Console, sess *client.Session, task *clientpb.Task, data []byte) (bool, error) {
	_, err := con.Rpc.AddScreenShot(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextScreenShot,
		Value: output.MarshalContext(&output.ScreenShotContext{
			Content: data,
		}),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterScreenshot(con *core.Console) {
	con.RegisterServerFunc("screenshots", func(con *core.Console) ([]*output.ScreenShotContext, error) {
		screenshots, err := GetScreenshots(con)
		if err != nil {
			return nil, err
		}
		return output.ToContexts[*output.ScreenShotContext](screenshots)
	}, nil)
	con.RegisterServerFunc("add_screenshot", AddScreenshot, nil)
}
