package build

import (
	"context"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
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
		con.Log.Info("No builders available\n")
	}
	return nil
}

func PrintArtifacts(builders *clientpb.Builders, con *repl.Console) error {
	var rowEntries []table.Row
	var row table.Row

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 15),
		table.NewColumn("Target", "Target", 30),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("Stager", "Stager", 10),
		table.NewColumn("Modules", "Modules", 30),
		table.NewColumn("Profile", "Profile", 20),
		table.NewColumn("Pipeline", "Pipeline", 20),
	}, false)
	for _, builder := range builders.Builders {
		row = table.NewRow(
			table.RowData{
				"Name":     builder.Name,
				"Target":   builder.Target,
				"Type":     builder.Type,
				"Stager":   builder.Stage,
				"Modules":  builder.Modules,
				"Profile":  builder.ProfileName,
				"Pipeline": builder.PipelineId,
			})

		rowEntries = append(rowEntries, row)
	}
	newTable := tui.NewModel(tableModel, nil, false, false)

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {
		downloadArtifactCallback(tableModel, newTable.Buffer, con)()
	})
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
	go func() {
		builder, err := DownloadArtifact(con, name)
		if err != nil {
			con.Log.Errorf("download artifact failed: %s", err)
			return
		}
		if builder.Type == consts.CommandBuildModules {
			builder.Name = builder.Name + consts.DllFile
		}
		if output == "" {
			output = filepath.Join(assets.GetTempDir(), builder.Name)
		}
		err = os.WriteFile(output, builder.Bin, 0644)
		if err != nil {
			con.Log.Errorf("open file failed: %s", err)
			return
		}
		con.Log.Infof("download artifact %s, save to %s\n", builder.Name, output)
	}()
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

func downloadArtifactCallback(tableModel *tui.TableModel, writer io.Writer, con *repl.Console) func() {
	selectRow := tableModel.GetHighlightedRow()
	if selectRow.Data == nil {
		return func() {
			con.Log.FErrorf(writer, "No row selected\n")
		}
	}
	return func() {
		go func() {
			builder, err := DownloadArtifact(con, selectRow.Data["Name"].(string))
			if err != nil {
				con.Log.Errorf("open file %s\n", err)
			}
			con.Log.Infof("download artifact %s\n", filepath.Join(assets.GetTempDir(), builder.Name))
			output := filepath.Join(assets.GetTempDir(), builder.Name)
			err = os.WriteFile(output, builder.Bin, 0644)
			if err != nil {
				con.Log.Errorf(err.Error() + "\n")
				return
			}
			return
		}()
	}
}
