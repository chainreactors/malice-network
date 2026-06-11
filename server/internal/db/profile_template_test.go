package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func writeProfileTemplate(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "implant.yaml"), []byte("basic:\n  name: "+name+"\n"), 0644); err != nil {
		t.Fatalf("write implant.yaml: %v", err)
	}
}

func TestRegisterProfileTemplates_IdempotentAndSkipsDeleted(t *testing.T) {
	initTestDB(t)

	sourceRoot := t.TempDir()
	writeProfileTemplate(t, sourceRoot, "template-a")
	writeProfileTemplate(t, sourceRoot, "template-b")

	first, err := RegisterProfileTemplates(sourceRoot)
	if err != nil {
		t.Fatalf("RegisterProfileTemplates first run failed: %v", err)
	}
	if first.Created != 2 || first.SkippedExisting != 0 || first.SkippedDeleted != 0 {
		t.Fatalf("first result = %+v, want 2 created", first)
	}

	second, err := RegisterProfileTemplates(sourceRoot)
	if err != nil {
		t.Fatalf("RegisterProfileTemplates second run failed: %v", err)
	}
	if second.Created != 0 || second.SkippedExisting != 2 || second.SkippedDeleted != 0 {
		t.Fatalf("second result = %+v, want 2 existing", second)
	}

	if err := DeleteProfileByName("template-a"); err != nil {
		t.Fatalf("DeleteProfileByName failed: %v", err)
	}

	third, err := RegisterProfileTemplates(sourceRoot)
	if err != nil {
		t.Fatalf("RegisterProfileTemplates third run failed: %v", err)
	}
	if third.Created != 0 || third.SkippedExisting != 1 || third.SkippedDeleted != 1 {
		t.Fatalf("third result = %+v, want 1 existing and 1 deleted", third)
	}

	active, err := GetProfiles()
	if err != nil {
		t.Fatalf("GetProfiles failed: %v", err)
	}
	if len(active) != 1 || active[0].Name != "template-b" {
		t.Fatalf("active profiles = %+v, want only template-b", active)
	}

	deleted, err := NewProfileQuery().Unscoped().WhereName("template-a").First()
	if err != nil {
		t.Fatalf("deleted template lookup failed: %v", err)
	}
	if !deleted.DeletedAt.Valid {
		t.Fatal("template-a should be soft-deleted")
	}
	if deleted.Source != models.ProfileSourceTemplate || deleted.SourceHash == "" {
		t.Fatalf("deleted template metadata = source:%q hash:%q", deleted.Source, deleted.SourceHash)
	}
}

func TestNewProfile_ReusesSoftDeletedProfileName(t *testing.T) {
	initTestDB(t)

	req := &clientpb.Profile{
		Name:          "recreate-me",
		ImplantConfig: []byte("basic:\n  name: recreate-me\n"),
	}
	if err := NewProfile(req); err != nil {
		t.Fatalf("NewProfile failed: %v", err)
	}
	created, err := GetProfileByName("recreate-me")
	if err != nil {
		t.Fatalf("GetProfileByName failed: %v", err)
	}

	if err := DeleteProfileByName("recreate-me"); err != nil {
		t.Fatalf("DeleteProfileByName failed: %v", err)
	}
	if err := NewProfile(req); err != nil {
		t.Fatalf("NewProfile recreate failed: %v", err)
	}

	recreated, err := GetProfileByName("recreate-me")
	if err != nil {
		t.Fatalf("GetProfileByName after recreate failed: %v", err)
	}
	if recreated.ID != created.ID {
		t.Fatalf("recreated ID = %s, want reused ID %s", recreated.ID, created.ID)
	}
	if recreated.DeletedAt.Valid {
		t.Fatal("recreated profile should be active")
	}
	if recreated.Source != models.ProfileSourceUser {
		t.Fatalf("recreated source = %q, want %q", recreated.Source, models.ProfileSourceUser)
	}

	var total int64
	if err := Session().Unscoped().Model(&models.Profile{}).Where("name = ?", "recreate-me").Count(&total).Error; err != nil {
		t.Fatalf("count profiles failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("profile rows = %d, want 1", total)
	}
}

func TestNewProfileStoresScopedPipelineListener(t *testing.T) {
	initTestDB(t)

	if _, err := SavePipeline(newTestPipeline("shared-profile-pipe", "listener-a")); err != nil {
		t.Fatalf("SavePipeline listener A failed: %v", err)
	}
	if _, err := SavePipeline(newTestPipeline("shared-profile-pipe", "listener-b")); err != nil {
		t.Fatalf("SavePipeline listener B failed: %v", err)
	}

	if err := NewProfile(&clientpb.Profile{
		Name:       "profile-listener-a",
		PipelineId: "listener-a:shared-profile-pipe",
	}); err != nil {
		t.Fatalf("NewProfile scoped pipeline failed: %v", err)
	}

	profile, err := GetProfileByName("profile-listener-a")
	if err != nil {
		t.Fatalf("GetProfileByName failed: %v", err)
	}
	if profile.PipelineID != "shared-profile-pipe" || profile.ListenerID != "listener-a" {
		t.Fatalf("stored profile pipeline = %q/%q, want shared-profile-pipe/listener-a", profile.PipelineID, profile.ListenerID)
	}

	pb := profile.ToProtobuf()
	if pb.PipelineId != "listener-a:shared-profile-pipe" {
		t.Fatalf("profile protobuf pipeline_id = %q, want scoped value", pb.PipelineId)
	}

	loaded, err := GetProfile("profile-listener-a")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if loaded == nil || loaded.Basic == nil || len(loaded.Basic.Targets) == 0 {
		t.Fatalf("loaded profile missing generated target: %#v", loaded)
	}
}

func TestNewProfileRejectsAmbiguousPipelineName(t *testing.T) {
	initTestDB(t)

	if _, err := SavePipeline(newTestPipeline("ambiguous-profile-pipe", "listener-a")); err != nil {
		t.Fatalf("SavePipeline listener A failed: %v", err)
	}
	if _, err := SavePipeline(newTestPipeline("ambiguous-profile-pipe", "listener-b")); err != nil {
		t.Fatalf("SavePipeline listener B failed: %v", err)
	}

	err := NewProfile(&clientpb.Profile{
		Name:       "profile-ambiguous",
		PipelineId: "ambiguous-profile-pipe",
	})
	if err == nil {
		t.Fatal("NewProfile should reject ambiguous pipeline names")
	}
	if !strings.Contains(err.Error(), "multiple pipelines named") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewProfileRejectsWrongScopedListener(t *testing.T) {
	initTestDB(t)

	if _, err := SavePipeline(newTestPipeline("scoped-profile-pipe", "listener-a")); err != nil {
		t.Fatalf("SavePipeline listener A failed: %v", err)
	}

	err := NewProfile(&clientpb.Profile{
		Name:       "profile-wrong-listener",
		PipelineId: "listener-b:scoped-profile-pipe",
	})
	if err == nil {
		t.Fatal("NewProfile should reject scoped pipeline with wrong listener")
	}
	if !strings.Contains(err.Error(), "record not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
