package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestUploadArtifact_RejectsEmptyBin(t *testing.T) {
	srv := &Server{}
	_, err := srv.UploadArtifact(context.Background(), &clientpb.Artifact{
		Name: "empty",
		Bin:  nil,
	})
	if err == nil {
		t.Fatal("expected error for empty binary, got nil")
	}
	if !strings.Contains(err.Error(), "empty binary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadArtifact_RejectsEmptySliceBin(t *testing.T) {
	srv := &Server{}
	_, err := srv.UploadArtifact(context.Background(), &clientpb.Artifact{
		Name: "empty-slice",
		Bin:  []byte{},
	})
	if err == nil {
		t.Fatal("expected error for empty binary, got nil")
	}
	if !strings.Contains(err.Error(), "empty binary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadArtifact_RejectsOversizedBin(t *testing.T) {
	srv := &Server{}
	// Allocate just over the limit. We only need the length to trigger the
	// check — the content doesn't matter and we don't want to actually
	// allocate 128 MiB in a unit test, so we test with a smaller slice and
	// temporarily lower the constant... except the constant is package-level.
	// Instead, create a slice of maxArtifactUploadSize+1 bytes. On modern
	// machines this is fine for a test (128 MiB + 1 byte).
	oversized := make([]byte, maxArtifactUploadSize+1)
	_, err := srv.UploadArtifact(context.Background(), &clientpb.Artifact{
		Name: "too-big",
		Bin:  oversized,
	})
	if err == nil {
		t.Fatal("expected error for oversized binary, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}
