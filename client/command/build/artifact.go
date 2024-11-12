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
	builders, err := con.Rpc.ListArtifact(context.Background(), &clientpb.Empty{})
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
	output, _ := cmd.Flags().GetString("output")
	builder, err := DownloadArtifact(con, name)
	if err != nil {
		return err
	}
	if output != "" {
		output = filepath.Join(assets.GetTempDir(), builder.Name)
	}
	err = os.WriteFile(output, builder.Bin, 0644)
	if err != nil {
		return err
	}
	con.Log.Infof("download artifact %s, save to %s\n", builder.Name, output)
	return nil
}

func DownloadArtifact(con *repl.Console, name string) (*clientpb.Builder, error) {
	return con.Rpc.DownloadArtifact(context.Background(), &clientpb.Builder{
		Name: name,
	})
}

func UploadArtifactCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	artifactType, _ := cmd.Flags().GetString("type")
	stage, _ := cmd.Flags().GetString("stage")
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = filepath.Base(path)
	}
	builder, err := UploadArtifact(con, path, name, artifactType, stage)
	if err != nil {
		return err
	}
	con.Log.Infof("upload artifact %s success, id:%d\n", builder.Name, builder.Id)
	return nil
}

func UploadArtifact(con *repl.Console, path string, name, artifactType, stage string) (*clientpb.Builder, error) {
	bin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return con.Rpc.UploadArtifact(context.Background(), &clientpb.Builder{
		Name:  name,
		Bin:   bin,
		Type:  artifactType,
		Stage: stage,
	})
}

func downloadArtifactCallback(tableModel *tui.TableModel, con *repl.Console) func() {
	selectRow := tableModel.GetSelectedRow()
	builder, err := DownloadArtifact(con, selectRow[0])
	if err != nil {
		return func() {
			con.Log.Errorf("open file %s", err)
		}
	}
	con.Log.Infof("download artifact %s\n", builder.Name)
	return func() {
		output := filepath.Join(assets.GetTempDir(), builder.Name)
		err = os.WriteFile(output, builder.Bin, 0644)
		if err != nil {
			con.Log.Errorf(err.Error() + "\n")
			return
		}
	}
}
