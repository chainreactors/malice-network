package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/formatutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/wabzsy/gonut"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ObjcopyPulse extracts shellcode from compiled artifact using objcopy
func ObjcopyPulse(builder *models.Artifact, platform, arch string) ([]byte, error) {
	absBuildOutputPath, err := filepath.Abs(configs.BuildOutputPath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve absolute path for build output directory '%s': %w", configs.BuildOutputPath, err)
	}

	// Create temporary file with unique name for objcopy output
	dstPath := filepath.Join(absBuildOutputPath, ".temp_objcopy_file")

	// Ensure cleanup of temporary file after processing
	defer func() {
		if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
			logs.Log.Warnf("Unable to cleanup temporary objcopy file '%s' - manual cleanup may be required: %v", dstPath, err)
		}
	}()

	// Prepare objcopy command to extract .text section as binary
	objcopyCommand := []string{"objcopy", "--only-section=.text", "-O", "binary", builder.Path, dstPath}
	logs.Log.Debugf("Executing objcopy command: %v", objcopyCommand)

	// Execute objcopy command with proper working directory
	cmd := exec.Command(objcopyCommand[0], objcopyCommand[1:]...)
	cmd.Dir = filepath.Dir(builder.Path)
	output, err := cmd.CombinedOutput()

	if len(output) > 0 {
		logs.Log.Debugf("Objcopy command output: %s", string(output))
	}

	if err != nil {
		return nil, fmt.Errorf("objcopy failed to extract shellcode from artifact '%s' (platform: %s, arch: %s): %w\nCommand: %v\nOutput: %s",
			builder.Name, platform, arch, err, objcopyCommand, string(output))
	}

	// Read the extracted binary shellcode
	bin, err := os.ReadFile(dstPath)
	if err != nil || len(bin) == 0 {
		return nil, fmt.Errorf("cannot read objcopy generated shellcode file '%s': %w", dstPath, err)
	}

	logs.Log.Infof("Successfully extracted %d bytes of shellcode from artifact '%s' using objcopy", len(bin), builder.Name)
	return bin, nil
}

func SRDIArtifact(artifactModel *models.Artifact, platform, arch string) ([]byte, error) {
	if !strings.Contains(artifactModel.Target, consts.Windows) {
		return []byte{}, errs.ErrPlartFormNotSupport
	}
	switch artifactModel.Type {
	case consts.CommandBuildPulse:
		return ObjcopyPulse(artifactModel, platform, arch)
	case consts.CommandBuildBeacon:
		return gonut.DonutShellcodeFromFile(artifactModel.Path, arch, "")
	default:
		return []byte{}, fmt.Errorf("unsupported artifact type: %s", artifactModel.Type)
	}
}

func (rpc *Server) DownloadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	artifactModel, err := db.GetArtifactByName(req.Name)
	if err != nil {
		return nil, err
	}
	bin, err := os.ReadFile(artifactModel.Path)
	if err != nil {
		return nil, err
	}
	if req.Format == "" || req.Format == "executable" {
		return artifactModel.ToArtifact(bin), nil
	} else {
		target, _ := consts.GetBuildTarget(artifactModel.Target)
		shellcodeBin, _ := SRDIArtifact(artifactModel, target.OS, target.Arch)
		formatter := formatutils.NewFormatter()

		if !formatter.IsSupported(req.Format) {
			return nil, fmt.Errorf("unsupported format: %s", req.Format)
		}
		result, _ := formatter.Convert(shellcodeBin, req.Format)
		return artifactModel.ToArtifact(result.Data), nil
	}
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	artifact, err := db.SaveArtifact(req.Name, req.Type, req.Platform, req.Arch, req.Stage, consts.ArtifactFromUpload)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(artifact.Path, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	return artifact.ToArtifact([]byte{}), nil
}

// for listener
func (rpc *Server) GetArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	artifact, err := db.GetArtifact(req)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(artifact.Path)
	if err != nil {
		return nil, err
	}
	return artifact.ToArtifact(data), nil
}

func (rpc *Server) ListArtifact(ctx context.Context, req *clientpb.Empty) (*clientpb.Artifacts, error) {
	artifacts, err := db.GetArtifacts()
	if err != nil {
		return nil, err
	}
	return artifacts, nil
}

func (rpc *Server) FindArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	return db.FindArtifact(req)
}

func (rpc *Server) DeleteArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, db.DeleteArtifactByName(req.Name)
}
