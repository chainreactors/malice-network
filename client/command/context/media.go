package context

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func GetMediaCmd(cmd *cobra.Command, con *core.Console) error {
	mediaContexts, err := GetMedia(con)
	if err != nil {
		return err
	}

	var rows []table.Row
	for _, ctx := range mediaContexts {
		media, err := output.ToContext[*output.MediaContext](ctx)
		if err != nil {
			return err
		}
		rows = append(rows, table.NewRow(table.RowData{
			"ID":      ctx.Id,
			"Session": ctx.Session.SessionId,
			"Task":    getTaskId(ctx.Task),
			"Kind":    media.MediaKind,
			"Name":    media.Name,
			"Path":    media.FilePath,
			"Size":    fmt.Sprintf("%.2f MB", float64(media.Size)/1024.0/1024.0),
		}))
	}

	model := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 36),
		table.NewColumn("Session", "Session", 16),
		table.NewColumn("Task", "Task", 8),
		table.NewColumn("Kind", "Kind", 12),
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Path", "Path", 40),
		table.NewColumn("Size", "Size", 12),
	}, true)
	model.SetRows(rows)
	con.Log.Console(model.View())
	return nil
}

func GetMedia(con *core.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextMedia)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func RegisterMedia(con *core.Console) {
	con.RegisterServerFunc("media_contexts", func(con *core.Console) ([]*output.MediaContext, error) {
		items, err := GetMedia(con)
		if err != nil {
			return nil, err
		}
		return output.ToContexts[*output.MediaContext](items)
	}, nil)
}
