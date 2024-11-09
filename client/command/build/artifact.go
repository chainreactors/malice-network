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

func ListArtifactCmd(cmd *cobra.Command, con *repl.Console) error {
	builders, err := con.Rpc.GetBuilders(context.Background(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(builders.Builders) > 0 {
		err = PrintArtifacts(builders, con)
		if err != nil {
			return err
		}
	} else {
		con.Log.Info("No builders available")
	}
	return nil
}

func PrintArtifacts(builders *clientpb.Builders, con *repl.Console) error {
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
		downloadArtifactCallback(tableModel, con)()
	})
	newTable := tui.NewModel(tableModel, tableModel.ConsoleHandler, true, false)
	err := newTable.Run()
	if err != nil {
		return err
	}
	tui.Reset()
	return nil
}

func DownloadArtifactCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := DownloadArtifact(con, name)
	if err != nil {
		return err
	}
	return nil
}

func UploadArtifactCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := UploadArtifact(con, name)
	if err != nil {
		return err
	}
	return nil
}
func DownloadArtifact(con *repl.Console, name string) (bool, error) {
	resp, err := con.Rpc.DownloadArtifact(context.Background(), &clientpb.Sync{
		FileId: name,
	})
	if err != nil {
		return false, err
	}
	filePath := filepath.Join(assets.GetTempDir(), resp.Name)
	err = os.WriteFile(filePath, resp.Content, 0644)
	if err != nil {
		return false, err
	}
	return true, nil
}

func UploadArtifact(con *repl.Console, name string) (bool, error) {
	bin, err := os.ReadFile(name)
	if err != nil {
		return false, err
	}

	_, err = con.Rpc.UploadArtifact(context.Background(), &clientpb.Bin{
		Name: name,
		Bin:  bin,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func downloadArtifactCallback(tableModel *tui.TableModel, con *repl.Console) func() {
	selectRow := tableModel.GetSelectedRow()
	_, err := DownloadArtifact(con, selectRow[0])
	if err != nil {
		return func() {
			con.Log.Errorf("open file %s", err)
		}
	}
	return func() {
	}
}
