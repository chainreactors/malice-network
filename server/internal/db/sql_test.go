package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(configs.ServerRootPath, 0700); err != nil {
		t.Fatalf("failed to create ServerRootPath %q: %v", configs.ServerRootPath, err)
	}
	dbFile := filepath.Join(configs.ServerRootPath, "malice.db")
	os.Remove(dbFile)
	os.Remove(dbFile + "-wal")
	os.Remove(dbFile + "-shm")
}

func TestNewDBClient_NilConfig(t *testing.T) {
	setupTestDB(t)

	client := NewDBClient(nil)
	if client == nil {
		t.Fatal("NewDBClient(nil) should return a valid client (defaulting to SQLite)")
	}
	if Adapter == nil {
		t.Fatal("Adapter should be initialized after NewDBClient")
	}
	if Adapter.Name() != "sqlite" {
		t.Errorf("expected sqlite adapter, got %q", Adapter.Name())
	}
}

func TestNewDBClient_EmptyDialect(t *testing.T) {
	setupTestDB(t)

	cfg := &configs.DatabaseConfig{
		Dialect: "",
	}
	client := NewDBClient(cfg)
	if client == nil {
		t.Fatal("NewDBClient with empty dialect should default to SQLite")
	}
	if Adapter.Name() != "sqlite" {
		t.Errorf("expected sqlite adapter for empty dialect, got %q", Adapter.Name())
	}
	// Verify dialect was corrected
	if cfg.Dialect != configs.Sqlite {
		t.Errorf("dialect should be corrected to %q, got %q", configs.Sqlite, cfg.Dialect)
	}
}

func TestNewDBClient_ExplicitSqlite(t *testing.T) {
	setupTestDB(t)

	cfg := &configs.DatabaseConfig{
		Dialect: configs.Sqlite,
	}
	client := NewDBClient(cfg)
	if client == nil {
		t.Fatal("NewDBClient with explicit sqlite should return valid client")
	}
	if Adapter.Name() != "sqlite" {
		t.Errorf("expected sqlite adapter, got %q", Adapter.Name())
	}
}

func TestNewDBClient_UnknownDialect(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("NewDBClient with unknown dialect should panic")
		}
	}()

	cfg := &configs.DatabaseConfig{
		Dialect: "oracle",
	}
	NewDBClient(cfg)
}

func TestNewDBClient_FixesInvalidPoolSettings(t *testing.T) {
	setupTestDB(t)

	cfg := &configs.DatabaseConfig{
		Dialect:      configs.Sqlite,
		MaxIdleConns: 0,
		MaxOpenConns: -1,
	}
	client := NewDBClient(cfg)
	if client == nil {
		t.Fatal("NewDBClient should return valid client")
	}
	if cfg.MaxIdleConns < 1 {
		t.Errorf("MaxIdleConns should be corrected to at least 1, got %d", cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns < 1 {
		t.Errorf("MaxOpenConns should be corrected to at least 1, got %d", cfg.MaxOpenConns)
	}
}

func TestNewDBClient_SetsGlobalAdapter(t *testing.T) {
	setupTestDB(t)

	Adapter = nil

	cfg := &configs.DatabaseConfig{
		Dialect: configs.Sqlite,
	}
	NewDBClient(cfg)

	if Adapter == nil {
		t.Fatal("global Adapter should be set after NewDBClient")
	}
}

func TestNewDBClient_PostgresInvalidDSN(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("NewDBClient with postgres config should panic when unable to connect")
		}
	}()

	cfg := &configs.DatabaseConfig{
		Dialect:  configs.Postgres,
		Host:     "invalid-host-that-does-not-exist.local",
		Port:     54321,
		Username: "nouser",
		Password: "nopass",
		Database: "nodb",
	}
	NewDBClient(cfg)
}
