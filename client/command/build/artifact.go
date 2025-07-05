package build

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/formatutils"
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
	artifacts, err := con.Rpc.ListArtifact(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(artifacts.Artifacts) > 0 {
		err = PrintArtifacts(artifacts, con)
		if err != nil {
			return err
		}
	} else {
		con.Log.Info("No artifacts available\n")
	}
	return nil
}

func PrintArtifacts(artifacts *clientpb.Artifacts, con *repl.Console) error {
	var rowEntries []table.Row
	var row table.Row

	defaultLengths := map[string]int{
		"ID":       6,
		"Name":     30,
		"Pipeline": 16,
		"Target":   22,
		"Type":     8,
		"Source":   7,
		//"Modules":   8,
		"CreatedAt": 20,
		"Profile":   18,
		"Status":    10,
	}

	for _, artifact := range artifacts.Artifacts {
		formattedTime := time.Unix(artifact.CreatedAt, 0).Format("2006-01-02 15:04:05")
		updateMaxLength(&defaultLengths, "ID", len(strconv.Itoa(int(artifact.Id))))
		updateMaxLength(&defaultLengths, "Name", len(artifact.Name))
		updateMaxLength(&defaultLengths, "Target", len(artifact.Target))
		updateMaxLength(&defaultLengths, "Type", len(artifact.Type))
		updateMaxLength(&defaultLengths, "Source", len(artifact.Source))
		//updateMaxLength(&defaultLengths, "Modules", len(artifact.))
		updateMaxLength(&defaultLengths, "Profile", len(artifact.Profile))
		updateMaxLength(&defaultLengths, "Pipeline", len(artifact.Pipeline))
		// updateMaxLength(&defaultLengths, "Time", len(formattedTime))
		pipelineDisplay := artifact.Pipeline
		if len(pipelineDisplay) > 16 {
			pipelineDisplay = pipelineDisplay[:13] + "..."
		}
		//nameDisplay := artifact.Name
		//if len(nameDisplay) > 20 {
		//	nameDisplay = nameDisplay[:17] + "..."
		//}
		profileDisplay := artifact.Profile
		if len(profileDisplay) > 18 {
			profileDisplay = profileDisplay[:15] + "..."
		}
		row = table.NewRow(
			table.RowData{
				"ID":     artifact.Id,
				"Name":   artifact.Name,
				"Type":   artifact.Type,
				"Target": artifact.Target,
				"Source": artifact.Source,
				//"Stager":   builder.Stage,
				//"Modules":   builder.Modules,
				"Profile":   profileDisplay,
				"Pipeline":  pipelineDisplay,
				"CreatedAt": formattedTime,
				"Status":    artifact.Status,
			})

		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", defaultLengths["ID"]),
		table.NewColumn("Name", "Name", defaultLengths["Name"]),
		table.NewColumn("Type", "Type", defaultLengths["Type"]),
		table.NewColumn("Pipeline", "Pipeline", defaultLengths["Pipeline"]),
		table.NewColumn("Target", "Target", defaultLengths["Target"]),
		table.NewColumn("Source", "Source", defaultLengths["Source"]),
		//table.NewColumn("Stager", "Stager", 10),
		//table.NewColumn("Modules", "Modules", defaultLengths["Modules"]),
		table.NewColumn("Profile", "Profile", defaultLengths["Profile"]),
		table.NewColumn("Status", "Status", defaultLengths["Status"]),
		table.NewColumn("CreatedAt", "CreatedAt", defaultLengths["CreatedAt"]),
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

	// Check if build status is completed before downloading
	status := selectRow.Data["Status"].(string)
	if status != consts.BuildStatusCompleted {
		con.Log.Errorf("Cannot download artifact: '%s' is not completed\n", selectRow.Data["Name"].(string))
		return nil
	}

	artifact, err := DownloadArtifact(con, selectRow.Data["Name"].(string), "")
	if err != nil {
		return err
	}
	fileExt, _ := fileutils.GetExtensionByBytes(artifact.Bin)
	con.Log.Infof("download artifact %s\n", filepath.Join(assets.GetTempDir(), artifact.Name+fileExt))
	output := filepath.Join(assets.GetTempDir(), artifact.Name+fileExt)
	err = os.WriteFile(output, artifact.Bin, 0644)
	if err != nil {
		return err
	}
	return nil
}

func ArtifactShowCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	artifact, err := con.Rpc.DownloadArtifact(con.Context(), &clientpb.Artifact{
		Name: name,
	})
	if err != nil {
		return err
	}
	printArtifact(artifact)

	showProfile, _ := cmd.Flags().GetBool("profile")
	if showProfile {
		con.Log.Console("full profile:\n\n")
		con.Log.Console(string(artifact.ProfileBytes))
	}

	return nil
}

func printArtifact(artifact *clientpb.Artifact) {
	art := map[string]interface{}{
		"ID":       artifact.Id,
		"Name":     artifact.Name,
		"Type":     artifact.Type,
		"Stage":    artifact.Stage,
		"Target":   artifact.Target,
		"Profile":  artifact.Profile,
		"Pipeline": artifact.Pipeline,
	}
	tui.RenderKV(art)
}

func DownloadArtifactCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	output, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")

	go func() {
		artifact, err := DownloadArtifact(con, name, format)
		if err != nil {
			con.Log.Errorf("Download artifact failed: %s", err)
			return
		}
		printArtifact(artifact)
		var fileExt string
		if format != "" && format != "executable" {
			formatter := formatutils.NewFormatter()
			if formatter.IsSupported(format) {
				fileExt = formatter.GetFormatExtension(format)
			} else {
				fileExt = ""
			}
		} else {
			fileExt, _ = fileutils.GetExtensionByBytes(artifact.Bin)
		}

		if output == "" {
			output = filepath.Join(assets.GetTempDir(), artifact.Name+fileExt)
		}

		err = os.WriteFile(output, artifact.Bin, 0644)
		if err != nil {
			con.Log.Errorf("Write file failed: %s", err)
			return
		}
		con.Log.Infof("Download artifact %s, save to %s\n", artifact.Name, output)
	}()
	return nil
}

func DownloadArtifact(con *repl.Console, name string, format string) (*clientpb.Artifact, error) {
	artifact, err := con.Rpc.DownloadArtifact(con.Context(), &clientpb.Artifact{
		Name:   name,
		Format: format,
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
	artifact, err := UploadArtifact(con, path, name, artifactType, stage)
	if err != nil {
		return err
	}
	con.Log.Infof("upload artifact %s success, id:%d\n", artifact.Name, artifact.Id)
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

func UploadArtifact(con *repl.Console, path string, name, artifactType, stage string) (*clientpb.Artifact, error) {
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
	artifactResp, err := con.Rpc.FindArtifact(con.Context(), &clientpb.Artifact{
		Arch:     arch,
		Platform: os,
		Type:     typ,
		Pipeline: pipeline,
		Format:   format,
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
