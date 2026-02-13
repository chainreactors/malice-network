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

func GetDownloadsCmd(cmd *cobra.Command, con *core.Console) error {
	downloads, err := GetDownloads(con)
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range downloads {
		download, err := output.ToContext[*output.DownloadContext](ctx)
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

func GetDownloads(con *core.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextDownload)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddDownload(con *core.Console, sess *client.Session, task *clientpb.Task, fileDesc *output.FileDescriptor) (bool, error) {
	_, err := con.Rpc.AddDownload(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextDownload,
		Value:   output.MarshalContext(&output.DownloadContext{FileDescriptor: fileDesc}),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterDownload(con *core.Console) {
	con.RegisterServerFunc("downloads", func(con *core.Console) ([]*output.DownloadContext, error) {
		downloads, err := GetDownloads(con)
		if err != nil {
			return nil, err
		}

		return output.ToContexts[*output.DownloadContext](downloads)
	}, nil)
	con.RegisterServerFunc("add_download", AddDownload, nil)
}
