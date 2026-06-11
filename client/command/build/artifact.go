package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	output2 "github.com/chainreactors/malice-network/helper/utils/output"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func updateMaxLength(maxLengths *map[string]int, key string, newLength int) {
	if (*maxLengths)[key] < newLength {
		(*maxLengths)[key] = newLength
	}
}

func ListArtifactCmd(cmd *cobra.Command, con *core.Console) error {
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

func PrintArtifacts(artifacts *clientpb.Artifacts, con *core.Console) error {
	var rowEntries []table.Row
	var row table.Row

	for _, artifact := range artifacts.Artifacts {
		formattedTime := time.Unix(artifact.CreatedAt, 0).Format("2006-01-02 15:04:05")
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
				//"Modules":   builder.Modules,
				"Profile":   profileDisplay,
				"Pipeline":  pipelineDisplay,
				"CreatedAt": formattedTime,
				"Status":    artifact.Status,
			})

		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 6),
		table.NewFlexColumn("Name", "Name", 1),
		table.NewColumn("Type", "Type", 10),
		table.NewFlexColumn("Pipeline", "Pipeline", 1),
		table.NewColumn("Target", "Target", 20),
		table.NewColumn("Source", "Source", 10),
		//table.NewColumn("Stager", "Stager", 10),
		//table.NewColumn("Modules", "Modules", defaultLengths["Modules"]),
		table.NewColumn("Profile", "Profile", 12),
		table.NewColumn("Status", "Status", 10),
		table.NewColumn("CreatedAt", "Created At", 16),
	}, common.ShouldUseStaticOutput(con))
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {})
	rendered, err := common.RunTable(con, tableModel)
	if err != nil {
		return err
	}
	if rendered {
		return nil
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
	err = WriteOriginArtifact(con, selectRow.Data["Name"].(string))
	if err != nil {
		return err
	}
	return nil
}

func ArtifactShowCmd(cmd *cobra.Command, con *core.Console) error {
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
		"Target":   artifact.Target,
		"Profile":  artifact.Profile,
		"Pipeline": artifact.Pipeline,
		"Size":     fileutils.Bytes(uint64(len(artifact.Bin))),
		"Comment":  artifact.Comment,
	}
	orderedKeys := []string{"ID", "Name", "Type", "Target", "Profile", "Pipeline", "Size", "Comment"}
	tui.RenderKVWithOptions(art, orderedKeys, tui.KVOptions{ShowHeader: true})
}

// Some optimization is needed.
func DownloadArtifactCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	output, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	rdi, _ := cmd.Flags().GetString("RDI")
	artifact, err := DownloadArtifact(con, name, format, rdi)
	if err != nil {
		con.Log.Errorf("Download artifact failed: %s", err)
		return err
	}
	printArtifact(artifact)
	go func() {
		if f, ok := output2.SupportedFormats[format]; ok && f.SupportRemote {
			var pipe *clientpb.Pipeline
			for _, pipeline := range con.Pipelines {
				if pipeline.Type == consts.WebsitePipeline {
					pipe = pipeline
					break
				}
			}

			usage := output2.SupportedFormats[format].Usage(pipe.URL() + output2.EncodeFormat(artifact.Name, format))
			con.Log.Infof("you can use this payload :\n--------\n%s\n--------\n", usage)
		} else {
			var fileExt string
			if format == consts.FormatExecutable && artifact.Format != "" {
				fileExt = artifact.Format
			} else if f, ok := output2.SupportedFormats[format]; ok {
				fileExt = f.Extension
			} else if artifact.Format != "" {
				fileExt = artifact.Format
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
		}

	}()
	return nil
}

func DownloadArtifact(con *core.Console, name string, format string, rdi string) (*clientpb.Artifact, error) {
	artifact, err := con.Rpc.DownloadArtifact(con.Context(), &clientpb.Artifact{
		Name:   name,
		Format: format,
		Rdi:    rdi,
	})
	if err != nil {
		return artifact, err
	}
	if len(artifact.Bin) == 0 {
		return artifact, errors.New("artifact maybe not download in server")
	}
	return artifact, err
}

func WriteOriginArtifact(con *core.Console, name string) error {
	artifact, err := DownloadArtifact(con, name, "", "")
	if err != nil {
		return err
	}
	fileExt := artifact.Format
	if fileExt == "" {
		fileExt, _ = fileutils.GetExtensionByBytes(artifact.Bin)
	}
	con.Log.Infof("download artifact %s\n", filepath.Join(assets.GetTempDir(), artifact.Name+fileExt))
	output := filepath.Join(assets.GetTempDir(), artifact.Name+fileExt)
	err = os.WriteFile(output, artifact.Bin, 0644)
	if err != nil {
		return err
	}
	return nil
}

func UploadArtifactCmd(cmd *cobra.Command, con *core.Console) error {
	path := cmd.Flags().Arg(0)
	artifactType, _ := cmd.Flags().GetString("type")
	name, _ := cmd.Flags().GetString("name")
	comment, _ := cmd.Flags().GetString("comment")
	target, _ := cmd.Flags().GetString("target")
	platform, _ := cmd.Flags().GetString("platform")
	arch, _ := cmd.Flags().GetString("arch")
	format, _ := cmd.Flags().GetString("format")
	if name == "" {
		name = filepath.Base(path)
	}
	// If --target is set but --platform/--arch were not, fall back to the
	// canonical OS/Arch derived from the build target table so server-side
	// metadata is consistent with the compiled-artifact path.
	if target != "" && (platform == "" || arch == "") {
		if t, ok := consts.GetBuildTarget(target); ok {
			if platform == "" {
				platform = t.OS
			}
			if arch == "" {
				arch = t.Arch
			}
		}
	}
	// Default Format to the source file's extension (.exe/.dll/...) so the
	// server doesn't have to guess from magic bytes on download.
	if format == "" {
		format = filepath.Ext(path)
	}
	artifact, err := UploadArtifact(con, path, name, artifactType, comment, target, platform, arch, format)
	if err != nil {
		return err
	}
	con.Log.Infof("upload artifact %s success, id:%d\n", artifact.Name, artifact.Id)
	return nil
}

func DeleteArtifactCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := DeleteArtifact(con, name)
	if err != nil {
		return err
	}

	con.Log.Infof("delete artifact %s success\n", name)
	return nil
}

// MaxArtifactUploadSize mirrors the server-side cap. Stat'ing first lets us
// reject huge files (e.g. wrong-path uploads of an ISO) before slurping them
// into memory and shipping them over the wire.
const MaxArtifactUploadSize = 128 << 20

func UploadArtifact(con *core.Console, path, name, artifactType, comment, target, platform, arch, format string) (*clientpb.Artifact, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("upload target %s is a directory", path)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("upload target %s is empty", path)
	}
	if info.Size() > MaxArtifactUploadSize {
		return nil, fmt.Errorf("artifact %s is %d bytes, exceeds limit of %d MiB",
			path, info.Size(), MaxArtifactUploadSize>>20)
	}
	bin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return con.Rpc.UploadArtifact(con.Context(), &clientpb.Artifact{
		Name:     name,
		Bin:      bin,
		Type:     artifactType,
		Comment:  comment,
		Target:   target,
		Platform: platform,
		Arch:     arch,
		Format:   format,
	})
}

func SearchArtifact(con *core.Console, pipeline, typ, format, os, arch string) (*clientpb.Artifact, error) {
	artifactResp, err := con.Rpc.FindArtifact(con.Context(), &clientpb.Artifact{
		Arch:     arch,
		Platform: os,
		Type:     typ,
		Pipeline: pipeline,
		Format:   format,
	})
	return artifactResp, err
}

func DeleteArtifact(con *core.Console, name string) (bool, error) {
	_, err := con.Rpc.DeleteArtifact(con.Context(), &clientpb.Artifact{
		Name: name,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
