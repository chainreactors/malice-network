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

	body, err := os.ReadFile(filepath.Join(configs.WebsitePath, "listener-a", "site-update", updated.ID.String()))
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

	contentPath := filepath.Join(configs.WebsitePath, "listener-a", "site-remove", content.ID.String())
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

func TestAddContentScopesSameWebsiteNameByListener(t *testing.T) {
	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	initTestDB(t)

	for _, listenerID := range []string{"listener-a", "listener-b"} {
		if _, err := SavePipeline(&models.Pipeline{
			Name:       "site-shared",
			ListenerId: listenerID,
			Type:       consts.WebsitePipeline,
			IP:         "127.0.0.1",
			Port:       8080,
			PipelineParams: &implanttypes.PipelineParams{
				WebPath: "/",
				Tls:     &implanttypes.TlsConfig{},
			},
		}); err != nil {
			t.Fatalf("SavePipeline(%s) failed: %v", listenerID, err)
		}
	}

	contentA, err := AddContent(&clientpb.WebContent{
		WebsiteId:  "site-shared",
		ListenerId: "listener-a",
		Path:       "/index.html",
		Type:       "raw",
		Content:    []byte("a"),
	})
	if err != nil {
		t.Fatalf("AddContent(listener-a) failed: %v", err)
	}
	contentB, err := AddContent(&clientpb.WebContent{
		WebsiteId:  "site-shared",
		ListenerId: "listener-b",
		Path:       "/index.html",
		Type:       "raw",
		Content:    []byte("b"),
	})
	if err != nil {
		t.Fatalf("AddContent(listener-b) failed: %v", err)
	}
	if contentA.ID == contentB.ID {
		t.Fatalf("content IDs should differ across listeners: %s", contentA.ID)
	}

	contentsA, err := FindWebContentsByWebsiteAndListener("site-shared", "listener-a")
	if err != nil {
		t.Fatalf("FindWebContentsByWebsiteAndListener(listener-a) failed: %v", err)
	}
	if len(contentsA) != 1 || contentsA[0].ListenerID != "listener-a" {
		t.Fatalf("listener-a contents = %#v, want one scoped row", contentsA)
	}
	contentsB, err := FindWebContentsByWebsiteAndListener("site-shared", "listener-b")
	if err != nil {
		t.Fatalf("FindWebContentsByWebsiteAndListener(listener-b) failed: %v", err)
	}
	if len(contentsB) != 1 || contentsB[0].ListenerID != "listener-b" {
		t.Fatalf("listener-b contents = %#v, want one scoped row", contentsB)
	}

	bodyA, err := os.ReadFile(filepath.Join(configs.WebsitePath, "listener-a", "site-shared", contentA.ID.String()))
	if err != nil {
		t.Fatalf("ReadFile(listener-a) failed: %v", err)
	}
	bodyB, err := os.ReadFile(filepath.Join(configs.WebsitePath, "listener-b", "site-shared", contentB.ID.String()))
	if err != nil {
		t.Fatalf("ReadFile(listener-b) failed: %v", err)
	}
	if string(bodyA) != "a" || string(bodyB) != "b" {
		t.Fatalf("content bodies = %q/%q, want a/b", bodyA, bodyB)
	}
}

func TestDeleteWebsiteByListenerKeepsSameNameOnOtherListener(t *testing.T) {
	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	initTestDB(t)

	for _, listenerID := range []string{"listener-a", "listener-b"} {
		if _, err := SavePipeline(&models.Pipeline{
			Name:       "site-delete-shared",
			ListenerId: listenerID,
			Type:       consts.WebsitePipeline,
			IP:         "127.0.0.1",
			Port:       8080,
			PipelineParams: &implanttypes.PipelineParams{
				WebPath: "/",
				Tls:     &implanttypes.TlsConfig{},
			},
		}); err != nil {
			t.Fatalf("SavePipeline(%s) failed: %v", listenerID, err)
		}
		if _, err := AddContent(&clientpb.WebContent{
			WebsiteId:  "site-delete-shared",
			ListenerId: listenerID,
			Path:       "/index.html",
			Type:       "raw",
			Content:    []byte(listenerID),
		}); err != nil {
			t.Fatalf("AddContent(%s) failed: %v", listenerID, err)
		}
	}

	if err := DeleteWebsiteByListener("site-delete-shared", "listener-a"); err != nil {
		t.Fatalf("DeleteWebsiteByListener failed: %v", err)
	}
	if _, err := FindPipelineByListener("site-delete-shared", "listener-a"); err == nil {
		t.Fatal("listener-a website should be deleted")
	}
	if _, err := FindPipelineByListener("site-delete-shared", "listener-b"); err != nil {
		t.Fatalf("listener-b website should remain: %v", err)
	}
	remaining, err := FindWebContentsByWebsiteAndListener("site-delete-shared", "listener-b")
	if err != nil {
		t.Fatalf("Find listener-b contents failed: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("listener-b content count = %d, want 1", len(remaining))
	}
	deleted, err := FindWebContentsByWebsiteAndListener("site-delete-shared", "listener-a")
	if err != nil {
		t.Fatalf("Find listener-a contents failed: %v", err)
	}
	if len(deleted) != 0 {
		t.Fatalf("listener-a content count = %d, want 0", len(deleted))
	}
}

func TestFindWebContentsByWebsiteAndListenerExcludesAmbiguousLegacyRows(t *testing.T) {
	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	initTestDB(t)

	for _, listenerID := range []string{"listener-a", "listener-b"} {
		if _, err := SavePipeline(&models.Pipeline{
			Name:       "site-legacy-shared",
			ListenerId: listenerID,
			Type:       consts.WebsitePipeline,
			IP:         "127.0.0.1",
			Port:       8080,
			PipelineParams: &implanttypes.PipelineParams{
				WebPath: "/",
				Tls:     &implanttypes.TlsConfig{},
			},
		}); err != nil {
			t.Fatalf("SavePipeline(%s) failed: %v", listenerID, err)
		}
	}
	if err := Session().Create(&models.WebsiteContent{
		PipelineID:  "site-legacy-shared",
		ListenerID:  "",
		Path:        "/legacy.txt",
		Type:        "raw",
		ContentType: "text/plain",
	}).Error; err != nil {
		t.Fatalf("Create legacy content failed: %v", err)
	}

	contents, err := FindWebContentsByWebsiteAndListener("site-legacy-shared", "listener-a")
	if err != nil {
		t.Fatalf("FindWebContentsByWebsiteAndListener failed: %v", err)
	}
	if len(contents) != 0 {
		t.Fatalf("scoped contents = %#v, want no ambiguous legacy rows", contents)
	}
}

func TestDeleteWebsiteByListenerDoesNotDeleteAmbiguousLegacyContent(t *testing.T) {
	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	initTestDB(t)

	for _, listenerID := range []string{"listener-a", "listener-b"} {
		if _, err := SavePipeline(&models.Pipeline{
			Name:       "site-delete-legacy-shared",
			ListenerId: listenerID,
			Type:       consts.WebsitePipeline,
			IP:         "127.0.0.1",
			Port:       8080,
			PipelineParams: &implanttypes.PipelineParams{
				WebPath: "/",
				Tls:     &implanttypes.TlsConfig{},
			},
		}); err != nil {
			t.Fatalf("SavePipeline(%s) failed: %v", listenerID, err)
		}
	}
	legacy := &models.WebsiteContent{
		PipelineID:  "site-delete-legacy-shared",
		ListenerID:  "",
		Path:        "/legacy.txt",
		Type:        "raw",
		ContentType: "text/plain",
	}
	if err := Session().Create(legacy).Error; err != nil {
		t.Fatalf("Create legacy content failed: %v", err)
	}

	if err := DeleteWebsiteByListener("site-delete-legacy-shared", "listener-a"); err != nil {
		t.Fatalf("DeleteWebsiteByListener failed: %v", err)
	}

	var count int64
	if err := Session().Model(&models.WebsiteContent{}).Where("id = ?", legacy.ID).Count(&count).Error; err != nil {
		t.Fatalf("Count legacy content failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("legacy content count = %d, want 1", count)
	}
}
