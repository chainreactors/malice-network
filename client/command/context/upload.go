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

func GetUploadsCmd(cmd *cobra.Command, con *core.Console) error {
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
			"Session": getSessionID(ctx.Session),
			"Task":    getTaskId(ctx.Task),
			"Name":    upload.Name,
			"Path":    upload.FilePath,
			"Size":    fmt.Sprintf("%.2f KB", float64(upload.Size)/1024),
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

func GetUploads(con *core.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextUpload)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddUpload(con *core.Console, sess *client.Session, task *clientpb.Task, fileDesc *output.FileDescriptor) (bool, error) {
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

func RegisterUpload(con *core.Console) {
	con.RegisterServerFunc("uploads", func(con *core.Console) ([]*output.UploadContext, error) {
		uploads, err := GetUploads(con)
		if err != nil {
			return nil, err
		}

		return output.ToContexts[*output.UploadContext](uploads)
	}, nil)
	con.RegisterServerFunc("add_upload", AddUpload, nil)
}
