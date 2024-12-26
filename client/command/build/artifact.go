package build

import (
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func ListArtifactCmd(cmd *cobra.Command, con *repl.Console) error {
	builders, err := con.Rpc.ListBuilder(con.Context(), &clientpb.Empty{})
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
		table.NewColumn("ID", "ID", 10),
		table.NewColumn("Name", "Name", 15),
		table.NewColumn("Pipeline", "Pipeline", 20),
		table.NewColumn("Target", "Target", 30),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("Source", "Source", 10),
		table.NewColumn("Stager", "Stager", 10),
		table.NewColumn("Modules", "Modules", 30),
		table.NewColumn("Time", "Time", 20),
		table.NewColumn("Profile", "Profile", 20),
	}, false)
	for _, builder := range builders.Builders {
		row = table.NewRow(
			table.RowData{
				"ID":       builder.Id,
				"Name":     builder.Name,
				"Target":   builder.Target,
				"Type":     builder.Type,
				"Source":   builder.Resource,
				"Stager":   builder.Stage,
				"Modules":  builder.Modules,
				"Profile":  builder.ProfileName,
				"Pipeline": builder.PipelineId,
				"Time":     builder.Time,
			})

		rowEntries = append(rowEntries, row)
	}
	newTable := tui.NewModel(tableModel, nil, false, false)

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {})
	err := newTable.Run()
	if err != nil {
		return err
	}

	tui.Reset()
	selectRow := tableModel.GetSelectedRow()
	if selectRow.Data == nil {
		con.Log.Error("No row selected\n")
		return nil
	}
	builder, err := DownloadArtifact(con, selectRow.Data["Name"].(string), false)
	if err != nil {
		con.Log.Errorf("open file %s\n", err)
	}
	con.Log.Infof("download artifact %s\n", filepath.Join(assets.GetTempDir(), builder.Name))
	output := filepath.Join(assets.GetTempDir(), builder.Name)
	err = os.WriteFile(output, builder.Bin, 0644)
	if err != nil {
		return err
	}
	return nil
}

func DownloadArtifactCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	output, _ := cmd.Flags().GetString("output")
	srdi, _ := cmd.Flags().GetBool("srdi")
	go func() {
		builder, err := DownloadArtifact(con, name, srdi)
		if err != nil {
			con.Log.Errorf("download artifact failed: %s", err)
			return
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

func DownloadArtifact(con *repl.Console, name string, srdi bool) (*clientpb.Artifact, error) {
	artifact, err := con.Rpc.DownloadArtifact(con.Context(), &clientpb.Artifact{
		Name:   name,
		IsSrdi: srdi,
	})
	if len(artifact.Bin) == 0 {
		return artifact, errors.New("artifact maybe not download in server")
	}
	return artifact, err
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

func DeleteArtifactCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := DeleteArtifact(con, name)
	if err != nil {
		return err
	}

	con.Log.Infof("delete artifact %s success\n", name)
	return nil
}

func UploadArtifact(con *repl.Console, path string, name, artifactType, stage string) (*clientpb.Builder, error) {
	bin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return con.Rpc.UploadArtifact(con.Context(), &clientpb.Artifact{
		Name:  name,
		Bin:   bin,
		Type:  artifactType,
		Stage: stage,
	})
}

func SearchArtifact(con *repl.Console, pipeline, typ, format, os, arch string) (*clientpb.Artifact, error) {
	var isSRDI bool
	switch format {
	case "srdi", "shellcode", "raw", "bin":
		isSRDI = true
	}

	return con.Rpc.FindArtifact(con.Context(), &clientpb.Artifact{
		Arch:     arch,
		Platform: os,
		Type:     typ,
		Pipeline: pipeline,
		IsSrdi:   isSRDI,
	})
}

func DeleteArtifact(con *repl.Console, name string) (bool, error) {
	_, err := con.Rpc.DeleteArtifact(con.Context(), &clientpb.Artifact{
		Name: name,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
