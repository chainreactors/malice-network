package db

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestAddContentUpdateReturnsLatestMetadataAndWritesFile(t *testing.T) {
	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	initTestDB(t)

	if _, err := SavePipeline(&models.Pipeline{
		Name:       "site-update",
		ListenerId: "listener-a",
		Type:       consts.WebsitePipeline,
		IP:         "127.0.0.1",
		Port:       8080,
		PipelineParams: &implanttypes.PipelineParams{
			WebPath: "/",
			Tls:     &implanttypes.TlsConfig{},
		},
	}); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}

	created, err := AddContent(&clientpb.WebContent{
		WebsiteId:   "site-update",
		Path:        "/index.html",
		Type:        "raw",
		ContentType: "text/plain",
		Content:     []byte("old"),
	})
	if err != nil {
		t.Fatalf("AddContent(create) failed: %v", err)
	}

	updatedBody := []byte("<h1>updated</h1>")
	updated, err := AddContent(&clientpb.WebContent{
		WebsiteId:   "site-update",
		Path:        "/index.html",
		Type:        "raw",
		ContentType: "text/html",
		Content:     updatedBody,
	})
	if err != nil {
		t.Fatalf("AddContent(update) failed: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("updated ID = %s, want %s", updated.ID, created.ID)
	}
	if updated.ContentType != "text/html" {
		t.Fatalf("updated content type = %q, want %q", updated.ContentType, "text/html")
	}
	if updated.Size != uint64(len(updatedBody)) {
		t.Fatalf("updated size = %d, want %d", updated.Size, len(updatedBody))
	}

	body, err := os.ReadFile(filepath.Join(configs.WebsitePath, "site-update", updated.ID.String()))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(body) != string(updatedBody) {
		t.Fatalf("website content = %q, want %q", string(body), string(updatedBody))
	}
}

func TestRemoveContentDeletesBackingFile(t *testing.T) {
	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	initTestDB(t)

	if _, err := SavePipeline(&models.Pipeline{
		Name:       "site-remove",
		ListenerId: "listener-a",
		Type:       consts.WebsitePipeline,
		IP:         "127.0.0.1",
		Port:       8080,
		PipelineParams: &implanttypes.PipelineParams{
			WebPath: "/",
			Tls:     &implanttypes.TlsConfig{},
		},
	}); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}

	content, err := AddContent(&clientpb.WebContent{
		WebsiteId: "site-remove",
		Path:      "/payload.bin",
		Type:      "raw",
		Content:   []byte("payload"),
	})
	if err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}

	contentPath := filepath.Join(configs.WebsitePath, "site-remove", content.ID.String())
	if _, err := os.Stat(contentPath); err != nil {
		t.Fatalf("expected content file to exist: %v", err)
	}

	if err := RemoveContent(content.ID.String()); err != nil {
		t.Fatalf("RemoveContent failed: %v", err)
	}
	if _, err := os.Stat(contentPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("content file stat error = %v, want not exist", err)
	}
	if _, err := FindWebContent(content.ID.String()); err == nil {
		t.Fatal("expected content record to be removed")
	}
}
