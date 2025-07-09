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
	//objcopyCommand := []string{"objcopy", "--only-section=.text", "-O", "binary", builder.Path, dstPath}
	objcopyCommand := []string{"objcopy", "-O", "binary", builder.Path, dstPath}
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
		return []byte{}, errs.ErrPlatFormNotSupport
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

	artifact, err := artifactModel.ToArtifact()
	if err != nil {
		return nil, err
	}

	return formatutils.ConvertArtifact(artifact, req.Format)
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	artifact, err := db.SaveArtifact(req.Name, req.Type, req.Platform, req.Arch, consts.ArtifactFromUpload)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(artifact.Path, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	return artifact.ToProtobuf([]byte{}), nil
}

// for listener
func (rpc *Server) GetArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	var artifactModel *models.Artifact
	var err error
	if req.Id == 0 {
		artifactModel, err = db.FindArtifactFromPipeline(req.Pipeline)
	} else {
		artifactModel, err = db.GetArtifactById(req.Id)
	}
	if err != nil {
		return nil, err
	}

	if artifactModel.Params != nil && artifactModel.Params.RelinkBeaconID != 0 {
		artifactModel, err = db.GetArtifactById(artifactModel.Params.RelinkBeaconID)
		if err != nil {
			return nil, err
		}
	}

	artifact, err := artifactModel.ToArtifact()
	if err != nil {
		return nil, err
	}

	return formatutils.ConvertArtifact(artifact, req.Format)
}

func (rpc *Server) ListArtifact(ctx context.Context, req *clientpb.Empty) (*clientpb.Artifacts, error) {
	artifacts, err := db.GetArtifacts()
	if err != nil {
		return nil, err
	}
	return artifacts, nil
}

func (rpc *Server) FindArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	artifact, err := db.FindArtifact(req)
	if err != nil {
		return nil, err
	}

	return formatutils.ConvertArtifact(artifact, req.Format)
}

func (rpc *Server) DeleteArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, db.DeleteArtifactByName(req.Name)
}
