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

func GetDownloadsCmd(cmd *cobra.Command, con *repl.Console) error {
	downloads, err := GetDownloads(con)
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range downloads {
		download, err := types.ToContext[*types.DownloadContext](ctx)
		if err != nil {
			return err
		}

		row := table.NewRow(table.RowData{
			"ID":      ctx.Id,
			"Session": ctx.Session.SessionId,
			"Task":    getTaskId(ctx.Task),
			"Name":    download.Name,
			"Path":    download.FilePath,
			"Size":    fmt.Sprintf("%.2f KB", float64(download.Size)/1024),
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

func GetDownloads(con *repl.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextDownload)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddDownload(con *repl.Console, sess *core.Session, task *clientpb.Task, fileDesc *types.FileDescriptor) (bool, error) {
	_, err := con.Rpc.AddDownload(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextDownload,
		Value:   types.MarshalContext(&types.DownloadContext{FileDescriptor: fileDesc}),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterDownload(con *repl.Console) {
	con.RegisterServerFunc("downloads", GetDownloads, nil)
	con.RegisterServerFunc("add_download", AddDownload, nil)
}
