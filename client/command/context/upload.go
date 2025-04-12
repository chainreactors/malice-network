package context

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/utils/output"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func GetUploadsCmd(cmd *cobra.Command, con *repl.Console) error {
	uploads, err := GetUploads(con)
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range uploads {
		upload, err := output.ToContext[*output.UploadContext](ctx)
		if err != nil {
			return err
		}

		row := table.NewRow(table.RowData{
			"ID":      ctx.Id,
			"Session": ctx.Session.SessionId,
			"Task":    getTaskId(ctx.Task),
			"Name":    upload.Name,
			"Path":    upload.FilePath,
			"Size":    fmt.Sprintf("%.2f KB", float64(upload.Size)/1024),
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
	con.Log.Console(tableModel.View())
	return nil
}

func GetUploads(con *repl.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextUpload)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddUpload(con *repl.Console, sess *core.Session, task *clientpb.Task, fileDesc *output.FileDescriptor) (bool, error) {
	_, err := con.Rpc.AddUpload(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextUpload,
		Value:   output.MarshalContext(&output.UploadContext{FileDescriptor: fileDesc}),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterUpload(con *repl.Console) {
	con.RegisterServerFunc("uploads", func(con *repl.Console) ([]*output.UploadContext, error) {
		uploads, err := GetUploads(con)
		if err != nil {
			return nil, err
		}

		return output.ToContexts[*output.UploadContext](uploads)
	}, nil)
	con.RegisterServerFunc("add_upload", AddUpload, nil)
}
