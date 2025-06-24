package build

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func updateMaxLength(maxLengths *map[string]int, key string, newLength int) {
	if (*maxLengths)[key] < newLength {
		(*maxLengths)[key] = newLength
	}
}

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

	defaultLengths := map[string]int{
		"ID":       6,
		"Pipeline": 16,
		"Target":   22,
		"Type":     8,
		"Stager":   10,
		"Source":   8,
		"Modules":  8,
		"Time":     20,
		"Profile":  20,
		"Status":   10,
	}

	for _, builder := range builders.Builders {
		formattedTime := time.Unix(builder.Time, 0).Format("2006-01-02 15:04:05")
		updateMaxLength(&defaultLengths, "ID", len(strconv.Itoa(int(builder.Id))))
		updateMaxLength(&defaultLengths, "Target", len(builder.Target))
		// updateMaxLength(&defaultLengths, "Type", len(builder.Type))
		// updateMaxLength(&defaultLengths, "Source", len(builder.Resource))
		updateMaxLength(&defaultLengths, "Modules", len(builder.Modules))
		updateMaxLength(&defaultLengths, "Profile", len(builder.ProfileName))
		updateMaxLength(&defaultLengths, "Pipeline", len(builder.Pipeline))
		// updateMaxLength(&defaultLengths, "Time", len(formattedTime))
		row = table.NewRow(
			table.RowData{
				"ID":     builder.Id,
				"Target": builder.Target,
				"Type":   builder.Type,
				"Source": builder.Resource,
				//"Stager":   builder.Stage,
				"Modules":  builder.Modules,
				"Profile":  builder.ProfileName,
				"Pipeline": builder.Pipeline,
				"Time":     formattedTime,
				"Status":   builder.Status,
			})

		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", defaultLengths["ID"]),
		table.NewColumn("Pipeline", "Pipeline", defaultLengths["Pipeline"]),
		table.NewColumn("Target", "Target", defaultLengths["Target"]),
		table.NewColumn("Type", "Type", defaultLengths["Type"]),
		table.NewColumn("Source", "Source", defaultLengths["Source"]),
		//table.NewColumn("Stager", "Stager", 10),
		table.NewColumn("Modules", "Modules", defaultLengths["Modules"]),
		table.NewColumn("Time", "Time", defaultLengths["Time"]),
		table.NewColumn("Profile", "Profile", defaultLengths["Profile"]),
		table.NewColumn("Status", "Status", defaultLengths["Status"]),
	}, false)
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {})
	err := tableModel.Run()
	if err != nil {
		return err
	}

	tui.Reset()
	selectRow := tableModel.GetSelectedRow()
	if selectRow.Data == nil {
		con.Log.Error("No row selected\n")
		return nil
	}
	builder, err := DownloadArtifact(con, selectRow.Data["ID"].(uint32), false)
	if err != nil {
		return err
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
	id := cmd.Flags().Arg(0)
	artifactID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	srdi, _ := cmd.Flags().GetBool("srdi")
	go func() {
		builder, err := DownloadArtifact(con, uint32(artifactID), srdi)
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

func DownloadArtifact(con *repl.Console, ID uint32, srdi bool) (*clientpb.Artifact, error) {
	artifact, err := con.Rpc.DownloadArtifact(con.Context(), &clientpb.Artifact{
		Id:     ID,
		IsSrdi: srdi,
	})
	if err != nil {
		return artifact, err
	}
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
	artifactResp, err := con.Rpc.FindArtifact(con.Context(), &clientpb.Artifact{
		Arch:     arch,
		Platform: os,
		Type:     typ,
		Pipeline: pipeline,
		IsSrdi:   isSRDI,
	})
	return artifactResp, err
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
