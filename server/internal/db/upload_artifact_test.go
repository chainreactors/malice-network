package db

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestSaveUploadedArtifact_SetsStatusCompleted(t *testing.T) {
	initTestDB(t)

	req := &clientpb.Artifact{
		Name:     "test-status",
		Type:     "beacon",
		Platform: "windows",
		Arch:     "x86_64",
		Target:   "x86_64-pc-windows-gnu",
		Format:   ".exe",
		Comment:  "unit test artifact",
	}

	artifact, err := SaveUploadedArtifact(req)
	if err != nil {
		t.Fatalf("SaveUploadedArtifact: %v", err)
	}
	if artifact.Status != consts.BuildStatusCompleted {
		t.Errorf("Status = %q, want %q", artifact.Status, consts.BuildStatusCompleted)
	}
}

func TestSaveUploadedArtifact_PreservesAllFields(t *testing.T) {
	initTestDB(t)

	req := &clientpb.Artifact{
		Name:     "test-fields",
		Type:     "modules",
		Platform: "linux",
		Arch:     "aarch64",
		Target:   "aarch64-unknown-linux-musl",
		Format:   ".so",
		Comment:  "my comment",
	}

	artifact, err := SaveUploadedArtifact(req)
	if err != nil {
		t.Fatalf("SaveUploadedArtifact: %v", err)
	}

	checks := []struct {
		field, got, want string
	}{
		{"Name", artifact.Name, req.Name},
		{"Type", artifact.Type, req.Type},
		{"Os", artifact.Os, req.Platform},
		{"Arch", artifact.Arch, req.Arch},
		{"Target", artifact.Target, req.Target},
		{"Format", artifact.Format, req.Format},
		{"Comment", artifact.Comment, req.Comment},
		{"Source", artifact.Source, consts.ArtifactFromUpload},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
	if artifact.Path == "" {
		t.Error("Path should not be empty")
	}
}

func TestSaveUploadedArtifact_DuplicateNameFails(t *testing.T) {
	initTestDB(t)

	req := &clientpb.Artifact{
		Name: "dup-name",
		Type: "beacon",
	}
	_, err := SaveUploadedArtifact(req)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = SaveUploadedArtifact(req)
	if err == nil {
		t.Fatal("expected error on duplicate name, got nil")
	}
}

func TestDeleteArtifactRow_RemovesRecord(t *testing.T) {
	initTestDB(t)

	req := &clientpb.Artifact{
		Name: "to-delete",
		Type: "pulse",
	}
	artifact, err := SaveUploadedArtifact(req)
	if err != nil {
		t.Fatalf("SaveUploadedArtifact: %v", err)
	}

	if err := DeleteArtifactRow(artifact.ID); err != nil {
		t.Fatalf("DeleteArtifactRow: %v", err)
	}

	found, err := GetArtifactByName("to-delete")
	if err == nil && found != nil {
		t.Error("expected artifact to be deleted, but it still exists")
	}
}

func TestSaveUploadedArtifact_FindableByStatus(t *testing.T) {
	initTestDB(t)

	req := &clientpb.Artifact{
		Name: "findable",
		Type: "beacon",
	}
	_, err := SaveUploadedArtifact(req)
	if err != nil {
		t.Fatalf("SaveUploadedArtifact: %v", err)
	}

	// SearchArtifact uses WhereStatus(BuildStatusCompleted) internally.
	// Verify the uploaded artifact is discoverable through that path.
	found, err := NewArtifactQuery().WhereName("findable").WhereStatus(consts.BuildStatusCompleted).First()
	if err != nil {
		t.Fatalf("query with status filter: %v", err)
	}
	if found == nil {
		t.Fatal("uploaded artifact not found with status=completed filter")
	}
}
