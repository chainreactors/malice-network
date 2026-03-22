package rpc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestResolveListenerIDPrefersExplicitFields(t *testing.T) {
	req := &clientpb.CtrlPipeline{
		Name:       "pipe-1",
		ListenerId: "listener-a",
		Pipeline: &clientpb.Pipeline{
			ListenerId: "listener-b",
		},
	}

	got, err := resolveListenerID(req)
	if err != nil {
		t.Fatalf("resolveListenerID returned error: %v", err)
	}
	if got != "listener-a" {
		t.Fatalf("resolveListenerID = %q, want %q", got, "listener-a")
	}
}

func TestResolveListenerIDFallsBackToPipelineListener(t *testing.T) {
	req := &clientpb.CtrlPipeline{
		Name: "pipe-1",
		Pipeline: &clientpb.Pipeline{
			ListenerId: "listener-b",
		},
	}

	got, err := resolveListenerID(req)
	if err != nil {
		t.Fatalf("resolveListenerID returned error: %v", err)
	}
	if got != "listener-b" {
		t.Fatalf("resolveListenerID = %q, want %q", got, "listener-b")
	}
}

func TestResolveListenerIDFallsBackToDatabaseLookup(t *testing.T) {
	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	if err := os.MkdirAll(configs.ServerRootPath, 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	oldDBClient := db.Client
	t.Cleanup(func() {
		db.Client = oldDBClient
	})
	var dbErr error
	db.Client, dbErr = db.NewDBClient(nil)
	if dbErr != nil {
		t.Fatalf("NewDBClient failed: %v", dbErr)
	}

	if _, err := db.SavePipeline(&models.Pipeline{
		Name:       "pipe-db",
		ListenerId: "listener-db",
		Type:       "tcp",
	}); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}

	got, err := resolveListenerID(&clientpb.CtrlPipeline{Name: "pipe-db"})
	if err != nil {
		t.Fatalf("resolveListenerID returned error: %v", err)
	}
	if got != "listener-db" {
		t.Fatalf("resolveListenerID = %q, want %q", got, "listener-db")
	}
}
