package build

import (
	"context"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

func DownloadCmd(cmd *cobra.Command, con *repl.Console) error {
	builders, err := con.Rpc.GetBuilders(context.Background(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(builders.Builders) > 0 {
		err = printBuilders(builders, con)
		if err != nil {
			return err
		}
	} else {
		con.Log.Info("No builders available")
	}
	return nil
}

func printBuilders(builders *clientpb.Builders, con *repl.Console) error {
	var rowEntries []table.Row
	var row table.Row

	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 15},
		{Title: "Target", Width: 15},
		{Title: "Type", Width: 10},
		{Title: "Stager", Width: 10},
		{Title: "Modules", Width: 30},
	}, false)
	for _, builder := range builders.Builders {
		row = table.Row{
			builder.Name,
			builder.Target,
			builder.Type,
			builder.Stager,
			strings.Join(builder.Modules, ","),
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {
		downloadBuilder(tableModel, con)()
	})
	newTable := tui.NewModel(tableModel, tableModel.ConsoleHandler, true, false)
	err := newTable.Run()
	if err != nil {
		return err
	}
	tui.Reset()
	return nil
}

func downloadBuilder(tableModel *tui.TableModel, con *repl.Console) func() {
	selectRow := tableModel.GetSelectedRow()
	resp, err := con.Rpc.DownloadOutput(context.Background(), &clientpb.Sync{
		FileId: selectRow[0],
	})
	if err != nil {
		return func() {
			con.Log.Errorf("download build output %s", err)
		}
	}
	filePath := filepath.Join(assets.GetTempDir(), resp.Name)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return func() {
			con.Log.Errorf("open file %s", err)
		}
	}
	defer file.Close()
	_, err = file.Write(resp.Content)
	if err != nil {
		return func() {
			con.Log.Errorf("write file %s", err)
		}
	}

	return func() {
	}
}
