package rpc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func TestUploadRejectsNilRequest(t *testing.T) {
	if _, err := (&Server{}).Upload(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("Upload(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestDownloadRejectsNilRequest(t *testing.T) {
	if _, err := (&Server{}).Download(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("Download(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestMergeChunksWritesCompleteOutput(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "1.chunk"), []byte("hello "), 0o600); err != nil {
		t.Fatalf("write chunk 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "2.chunk"), []byte("world"), 0o600); err != nil {
		t.Fatalf("write chunk 2: %v", err)
	}

	finalPath := filepath.Join(t.TempDir(), "downloads", "file.bin")
	if err := mergeChunks(tempDir, finalPath, 2); err != nil {
		t.Fatalf("mergeChunks failed: %v", err)
	}

	content, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("read merged file: %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("merged content = %q, want hello world", string(content))
	}
}

func TestMergeChunksMissingChunkDoesNotLeavePartialOutput(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "1.chunk"), []byte("partial"), 0o600); err != nil {
		t.Fatalf("write chunk 1: %v", err)
	}

	finalPath := filepath.Join(t.TempDir(), "downloads", "file.bin")
	err := mergeChunks(tempDir, finalPath, 2)
	if err == nil || !strings.Contains(err.Error(), "failed to read chunk 2") {
		t.Fatalf("mergeChunks error = %v, want missing chunk error", err)
	}
	if _, statErr := os.Stat(finalPath); !os.IsNotExist(statErr) {
		t.Fatalf("final output should not exist after failed merge, stat err = %v", statErr)
	}
}

func TestUploadRequestTypeReferenceCompiles(t *testing.T) {
	_ = &implantpb.UploadRequest{}
}
