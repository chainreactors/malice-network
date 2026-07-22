package rpc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
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

func TestDownloadChunkCount(t *testing.T) {
	cases := []struct {
		size      int
		chunkSize int
		want      int
	}{
		{size: 0, chunkSize: 512, want: 0},
		{size: 1, chunkSize: 512, want: 1},
		{size: 512, chunkSize: 512, want: 1},
		{size: 513, chunkSize: 512, want: 2},
		{size: 1024, chunkSize: 512, want: 2},
		{size: 1025, chunkSize: 512, want: 3},
		{size: 10, chunkSize: 0, want: 0},
	}

	for _, tc := range cases {
		if got := downloadChunkCount(tc.size, tc.chunkSize); got != tc.want {
			t.Fatalf("downloadChunkCount(%d, %d) = %d, want %d", tc.size, tc.chunkSize, got, tc.want)
		}
	}
}

func TestScanDownloadChunks(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "1.chunk"), []byte("a"), 0o600); err != nil {
		t.Fatalf("write chunk 1: %v", err)
	}

	cur, complete, err := scanDownloadChunks(tempDir, 2)
	if err != nil {
		t.Fatalf("scanDownloadChunks failed: %v", err)
	}
	if complete {
		t.Fatal("expected incomplete chunk set")
	}
	if cur != 2 {
		t.Fatalf("resume cursor = %d, want 2", cur)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "2.chunk"), []byte("b"), 0o600); err != nil {
		t.Fatalf("write chunk 2: %v", err)
	}
	cur, complete, err = scanDownloadChunks(tempDir, 2)
	if err != nil {
		t.Fatalf("scanDownloadChunks complete failed: %v", err)
	}
	if !complete {
		t.Fatal("expected complete chunk set")
	}
	if cur != 2 {
		t.Fatalf("complete cursor = %d, want 2", cur)
	}
}

func TestFinalizeDownloadHandlesContextSaveFailureGracefully(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-download-finalize", "rpc-download-pipe", true)
	task := sess.NewTask("download", 1)
	t.Cleanup(task.Close)

	content := []byte("hello download")
	tempDir := filepath.Join(configs.TempPath, "downloads-test")
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatalf("mkdir temp dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "1.chunk"), content, 0o600); err != nil {
		t.Fatalf("write chunk: %v", err)
	}
	finalPath := filepath.Join(t.TempDir(), "out.bin")
	checksum, err := fileutils.CalculateSHA256Checksum(filepath.Join(tempDir, "1.chunk"))
	if err != nil {
		t.Fatalf("checksum chunk: %v", err)
	}

	origSave := rpcFileSaveContext
	rpcFileSaveContext = func(*clientpb.Context) (*models.Context, error) {
		return nil, errors.New("save context failed")
	}
	t.Cleanup(func() {
		rpcFileSaveContext = origSave
	})

	greq := &GenericRequest{Task: task, Session: sess}
	req := &implantpb.DownloadRequest{Name: "report.txt", Path: "C:/Temp/report.txt"}
	resp := &implantpb.Spite{
		TaskId: task.Id,
		Body: &implantpb.Spite_DownloadResponse{
			DownloadResponse: &implantpb.DownloadResponse{
				Checksum: checksum,
				Size:     uint64(len(content)),
				Cur:      1,
				Content:  content,
			},
		},
	}

	if err := finalizeDownload(greq, req, resp, resp.GetDownloadResponse(), 1, finalPath, tempDir); err != nil {
		t.Fatalf("finalizeDownload failed: %v", err)
	}
	if !task.Finished() {
		t.Fatal("task should finish even when context save fails")
	}
	if task.FinishedAtTime().IsZero() {
		t.Fatal("task finished time should be set")
	}
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("final output missing: %v", err)
	}
}

func TestUploadRequestTypeReferenceCompiles(t *testing.T) {
	_ = &implantpb.UploadRequest{}
}

func TestUploadBlockCount(t *testing.T) {
	cases := []struct {
		totalSize    int64
		packetLength int
		want         int
	}{
		{0, 512, 1}, // zero-byte needs one final empty block
		{1, 512, 1},
		{512, 512, 1},
		{513, 512, 2},
		{1024, 512, 2},
		{1025, 512, 3},
		{10, 0, 1}, // no packet limit → single logical block
		{-1, 512, 1},
	}
	for _, tc := range cases {
		if got := uploadBlockCount(tc.totalSize, tc.packetLength); got != tc.want {
			t.Fatalf("uploadBlockCount(%d, %d) = %d, want %d", tc.totalSize, tc.packetLength, got, tc.want)
		}
	}
}

func TestShouldInlineUpload(t *testing.T) {
	cases := []struct {
		totalSize    int64
		packetLength int
		want         bool
	}{
		{0, 512, false}, // empty must stream so implant gets final Block
		{1, 512, true},
		{512, 512, true},
		{513, 512, false},
		{10, 0, true},
		{-1, 512, false},
	}
	for _, tc := range cases {
		if got := shouldInlineUpload(tc.totalSize, tc.packetLength); got != tc.want {
			t.Fatalf("shouldInlineUpload(%d, %d) = %v, want %v", tc.totalSize, tc.packetLength, got, tc.want)
		}
	}
}

func TestCloneUploadMetadataPreservesOverride(t *testing.T) {
	src := &implantpb.UploadRequest{
		Name:     "payload.bin",
		Target:   "C:\\Windows\\Temp\\payload.bin",
		Priv:     0o644,
		Data:     []byte("should-not-copy"),
		Hidden:   true,
		Override: true,
	}
	got := cloneUploadMetadata(src)
	if got.Name != src.Name || got.Target != src.Target || got.Priv != src.Priv || got.Hidden != src.Hidden {
		t.Fatalf("metadata fields not preserved: %+v", got)
	}
	if !got.Override {
		t.Fatal("Override was dropped from multi-block metadata clone")
	}
	if len(got.Data) != 0 {
		t.Fatalf("Data should be empty in metadata-only clone, got %d bytes", len(got.Data))
	}
	// nil-safe
	empty := cloneUploadMetadata(nil)
	if empty == nil {
		t.Fatal("cloneUploadMetadata(nil) returned nil")
	}
}
