package rpc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
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

func TestUpdateArtifact_UpdatesComment(t *testing.T) {
	newRPCTestEnv(t)
	srv := &Server{}

	artifact, err := db.SaveUploadedArtifact(&clientpb.Artifact{
		Name:    "rpc-comment",
		Type:    "beacon",
		Comment: "old",
	})
	if err != nil {
		t.Fatalf("SaveUploadedArtifact: %v", err)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	updated, err := srv.UpdateArtifact(context.Background(), &clientpb.Artifact{
		Id:      artifact.ID,
		Comment: "new rpc comment",
	})
	if err != nil {
		t.Fatalf("UpdateArtifact: %v", err)
	}
	if updated.Comment != "new rpc comment" {
		t.Fatalf("updated comment = %q, want %q", updated.Comment, "new rpc comment")
	}

	found, err := db.GetArtifactByName(artifact.Name)
	if err != nil {
		t.Fatalf("GetArtifactByName: %v", err)
	}
	if found.Comment != "new rpc comment" {
		t.Fatalf("stored comment = %q, want %q", found.Comment, "new rpc comment")
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlArtifactUpdate)
	if event.EventType != consts.EventBuild || event.Job.GetName() != artifact.Name {
		t.Fatalf("unexpected artifact update event: %#v", event)
	}
}

func TestDeleteArtifactPublishesLifecycleEvent(t *testing.T) {
	newRPCTestEnv(t)
	artifact, err := db.SaveUploadedArtifact(&clientpb.Artifact{Name: "rpc-delete-event", Type: "beacon"})
	if err != nil {
		t.Fatalf("SaveUploadedArtifact: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(artifact.Path), 0o700); err != nil {
		t.Fatalf("MkdirAll artifact path: %v", err)
	}
	if err := os.WriteFile(artifact.Path, []byte("artifact"), 0o600); err != nil {
		t.Fatalf("WriteFile artifact: %v", err)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	if _, err := (&Server{}).DeleteArtifact(context.Background(), &clientpb.Artifact{Name: artifact.Name}); err != nil {
		t.Fatalf("DeleteArtifact: %v", err)
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlArtifactDelete)
	if event.EventType != consts.EventBuild || event.Job.GetName() != artifact.Name {
		t.Fatalf("unexpected artifact delete event: %#v", event)
	}
}

func TestUpdateArtifact_RejectsMissingSelector(t *testing.T) {
	newRPCTestEnv(t)
	srv := &Server{}

	_, err := srv.UpdateArtifact(context.Background(), &clientpb.Artifact{
		Comment: "new",
	})
	if err == nil {
		t.Fatal("expected missing selector error, got nil")
	}
}
