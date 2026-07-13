package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const artifactStreamChunkSize int64 = 512 * 1024

// ObjcopyPulse extracts shellcode from compiled artifact using objcopy
func ObjcopyPulse(builder *models.Artifact, platform, arch string) ([]byte, error) {
	absBuildOutputPath, err := filepath.Abs(configs.TempPath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve absolute path for build output directory '%s': %w", configs.TempPath, err)
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

func (rpc *Server) DownloadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	artifactModel, err := db.GetArtifactByName(req.Name)
	if err != nil {
		return nil, err
	}

	artifact, err := artifactModel.ToArtifact()
	if err != nil {
		return nil, err
	}

	return build.ConvertArtifact(artifact, req.Format, req.Rdi)
}

// DownloadArtifactStream streams an artifact without loading the original
// binary into memory when no output conversion is requested. Converted
// formats preserve DownloadArtifact semantics and are emitted as chunks after
// conversion completes.
func (rpc *Server) DownloadArtifactStream(req *clientpb.Artifact, stream clientrpc.MaliceRPC_DownloadArtifactStreamServer) error {
	if req == nil || req.Name == "" {
		return fmt.Errorf("artifact name is required")
	}

	artifactModel, err := db.GetArtifactByName(req.Name)
	if err != nil {
		return err
	}

	if req.Format == "" || req.Format == consts.FormatExecutable {
		file, err := os.Open(artifactModel.Path)
		if err != nil {
			return err
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return err
		}
		return sendArtifactContentStream(artifactModel.ToProtobuf(nil), stat.Size(), file, stream)
	}

	artifact, err := artifactModel.ToArtifact()
	if err != nil {
		return err
	}
	artifact, err = build.ConvertArtifact(artifact, req.Format, req.Rdi)
	if err != nil {
		return err
	}
	content := artifact.Bin
	artifact.Bin = nil
	return sendArtifactContentStream(artifact, int64(len(content)), bytes.NewReader(content), stream)
}

func sendArtifactContentStream(
	header *clientpb.Artifact,
	totalSize int64,
	reader io.Reader,
	stream clientrpc.MaliceRPC_DownloadArtifactStreamServer,
) error {
	if header == nil {
		return fmt.Errorf("artifact stream header is required")
	}
	header.Bin = nil
	if totalSize <= 0 {
		return stream.Send(&clientpb.ArtifactChunk{
			Header:    header,
			TotalSize: 0,
			Eof:       true,
		})
	}

	if err := stream.Send(&clientpb.ArtifactChunk{
		Header:    header,
		TotalSize: totalSize,
		Eof:       false,
	}); err != nil {
		return err
	}

	for offset := int64(0); offset < totalSize; {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}

		chunkSize := artifactStreamChunkSize
		if remaining := totalSize - offset; remaining < chunkSize {
			chunkSize = remaining
		}
		chunk := make([]byte, chunkSize)
		n, readErr := reader.Read(chunk)
		if n == 0 {
			if readErr == nil {
				return io.ErrNoProgress
			}
			if errors.Is(readErr, io.EOF) {
				return fmt.Errorf("read artifact content at offset %d: %w", offset, io.ErrUnexpectedEOF)
			}
			return readErr
		}

		nextOffset := offset + int64(n)
		if err := stream.Send(&clientpb.ArtifactChunk{
			Content:   chunk[:n],
			Offset:    offset,
			TotalSize: totalSize,
			Eof:       nextOffset == totalSize,
		}); err != nil {
			return err
		}
		offset = nextOffset

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				if offset < totalSize {
					return fmt.Errorf("read artifact content at offset %d: %w", offset, io.ErrUnexpectedEOF)
				}
				return nil
			}
			return readErr
		}
	}
	return nil
}

// maxArtifactUploadSize caps user-uploaded artifact binaries. 128 MiB covers
// debug builds and OLLVM-obfuscated implants with several times the headroom
// of a typical release beacon (~5 MiB), while still rejecting accidental ISO/
// VM-image uploads that would otherwise exhaust disk.
const maxArtifactUploadSize = 128 << 20

func publishArtifactLifecycleEvent(operation string, artifact *clientpb.Artifact) {
	if artifact == nil {
		return
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventBuild,
		Op:        operation,
		Job:       &clientpb.Job{Name: artifact.Name},
		Important: true,
		Message:   fmt.Sprintf("artifact %s changed", artifact.Name),
	})
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	if len(req.Bin) == 0 {
		return nil, fmt.Errorf("uploaded artifact has empty binary")
	}
	if len(req.Bin) > maxArtifactUploadSize {
		return nil, fmt.Errorf("uploaded artifact is %d bytes, exceeds limit of %d bytes (%d MiB)",
			len(req.Bin), maxArtifactUploadSize, maxArtifactUploadSize>>20)
	}
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	// Reject duplicate names up-front so the caller gets a clear error instead
	// of the raw "UNIQUE constraint failed" surfaced by GORM.
	if existing, _ := db.GetArtifactByName(req.Name); existing != nil {
		return nil, fmt.Errorf("artifact %q already exists (id=%d), use a different --name or delete the existing one",
			req.Name, existing.ID)
	}
	if req.Format == "" {
		if ext, err := fileutils.GetExtensionByBytes(req.Bin); err == nil {
			req.Format = ext
		}
	}
	artifact, err := db.SaveUploadedArtifact(req)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(artifact.Path, req.Bin, 0644); err != nil {
		// Roll back the DB row so the table doesn't carry a record whose
		// backing file never landed on disk.
		if delErr := db.DeleteArtifactRow(artifact.ID); delErr != nil {
			logs.Log.Warnf("failed to rollback artifact %d after write failure: %v", artifact.ID, delErr)
		}
		return nil, fmt.Errorf("write artifact %s: %w", req.Name, err)
	}
	result := artifact.ToProtobuf([]byte{})
	publishArtifactLifecycleEvent(consts.CtrlArtifactUpload, result)
	return result, nil
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

	return build.ConvertArtifact(artifact, req.Format, "")
}

func (rpc *Server) ListArtifact(ctx context.Context, req *clientpb.Empty) (*clientpb.Artifacts, error) {
	modelArtifacts, err := db.ListArtifacts()
	if err != nil {
		return nil, err
	}
	return modelArtifacts.ToProtobuf(), nil
}

func (rpc *Server) UpdateArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	artifact, err := db.UpdateArtifactComment(req)
	if err != nil {
		return nil, err
	}
	result := artifact.ToProtobuf([]byte{})
	publishArtifactLifecycleEvent(consts.CtrlArtifactUpdate, result)
	return result, nil
}

func (rpc *Server) FindArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	artifact, err := db.FindArtifact(req, req.Format != "null")
	if err != nil {
		return nil, err
	}

	return build.ConvertArtifact(artifact, req.Format, "")
}

func (rpc *Server) DeleteArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Empty, error) {
	if err := db.DeleteArtifactByName(req.Name); err != nil {
		return nil, err
	}
	publishArtifactLifecycleEvent(consts.CtrlArtifactDelete, req)
	return &clientpb.Empty{}, nil
}
