package db

import (
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm"
)

// allModels mirrors the model list used in NewDBClient.
var testAllModels = []interface{}{
	&models.Pipeline{},
	&models.Operator{},
	&models.Certificate{},
	&models.AuthzRule{},
	&models.Profile{},
	&models.WebsiteContent{},
	&models.Session{},
	&models.Artifact{},
	&models.Task{},
	&models.Context{},
}

// openTestSQLite creates a fresh in-memory SQLite database using the project's
// custom dialector (not gorm.io/driver/sqlite).
func openTestSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	return db
}

// TestSQLiteAutoMigrate_Fresh verifies that AutoMigrate succeeds on a fresh database.
func TestSQLiteAutoMigrate_Fresh(t *testing.T) {
	db := openTestSQLite(t)

	if err := db.AutoMigrate(testAllModels...); err != nil {
		t.Fatalf("first AutoMigrate failed on fresh database: %v", err)
	}

	// Verify all tables were created
	for _, table := range []string{"pipelines", "operators", "certificates", "authz_rules",
		"profiles", "website_contents", "sessions", "artifacts", "tasks", "contexts"} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("table %q should exist after AutoMigrate", table)
		}
	}
}

// TestSQLiteAutoMigrate_Idempotent verifies that running AutoMigrate twice
// does NOT produce "table already exists" or any other error.
func TestSQLiteAutoMigrate_Idempotent(t *testing.T) {
	db := openTestSQLite(t)

	if err := db.AutoMigrate(testAllModels...); err != nil {
		t.Fatalf("first AutoMigrate failed: %v", err)
	}

	// Second run — this is the one that used to fail with
	// "table `pipelines` already exists" or "near ALTER: syntax error"
	// or "near CONSTRAINT: syntax error"
	if err := db.AutoMigrate(testAllModels...); err != nil {
		t.Fatalf("second AutoMigrate failed (should be idempotent): %v", err)
	}
}

// TestSQLiteAutoMigrate_ThirdRun ensures stability across multiple restarts.
func TestSQLiteAutoMigrate_ThirdRun(t *testing.T) {
	db := openTestSQLite(t)

	for i := 1; i <= 3; i++ {
		if err := db.AutoMigrate(testAllModels...); err != nil {
			t.Fatalf("AutoMigrate run %d failed: %v", i, err)
		}
	}
}

// TestSQLiteHasTable verifies the custom HasTable implementation.
func TestSQLiteHasTable(t *testing.T) {
	db := openTestSQLite(t)

	if db.Migrator().HasTable("nonexistent_table") {
		t.Error("HasTable should return false for nonexistent table")
	}

	db.Exec("CREATE TABLE test_has_table (id INTEGER PRIMARY KEY)")

	if !db.Migrator().HasTable("test_has_table") {
		t.Error("HasTable should return true for existing table")
	}
}

// TestSQLiteHasColumn verifies the custom HasColumn implementation.
func TestSQLiteHasColumn(t *testing.T) {
	db := openTestSQLite(t)

	db.Exec("CREATE TABLE test_has_col (id INTEGER PRIMARY KEY, name TEXT)")

	if !db.Migrator().HasColumn("test_has_col", "name") {
		t.Error("HasColumn should return true for existing column")
	}
	if db.Migrator().HasColumn("test_has_col", "nonexistent") {
		t.Error("HasColumn should return false for nonexistent column")
	}
}

// TestSQLiteHasIndex verifies the custom HasIndex implementation.
func TestSQLiteHasIndex(t *testing.T) {
	db := openTestSQLite(t)

	db.Exec("CREATE TABLE test_has_idx (id INTEGER PRIMARY KEY, name TEXT)")
	db.Exec("CREATE INDEX idx_test_name ON test_has_idx(name)")

	if !db.Migrator().HasIndex("test_has_idx", "idx_test_name") {
		t.Error("HasIndex should return true for existing index")
	}
	if db.Migrator().HasIndex("test_has_idx", "idx_nonexistent") {
		t.Error("HasIndex should return false for nonexistent index")
	}
}

// TestSQLiteAutoMigrate_DisableFK verifies AutoMigrate works with
// DisableForeignKeyConstraintWhenMigrating (same strategy as PostgreSQL path).
func TestSQLiteAutoMigrate_DisableFK(t *testing.T) {
	db := openTestSQLite(t)
	db.DisableForeignKeyConstraintWhenMigrating = true

	if err := db.AutoMigrate(testAllModels...); err != nil {
		t.Fatalf("first AutoMigrate with DisableFK failed: %v", err)
	}
	if err := db.AutoMigrate(testAllModels...); err != nil {
		t.Fatalf("second AutoMigrate with DisableFK failed: %v", err)
	}
}

// TestNewDBClient_AutoMigrateIdempotent tests via the full NewDBClient path.
func TestNewDBClient_AutoMigrateIdempotent(t *testing.T) {
	setupTestDB(t)

	cfg := &configs.DatabaseConfig{Dialect: configs.Sqlite}

	// First init
	client1, err := NewDBClient(cfg)
	if err != nil {
		t.Fatalf("first NewDBClient failed: %v", err)
	}
	if client1 == nil {
		t.Fatal("first NewDBClient should succeed")
	}

	// Second init (simulates server restart) — must not error
	client2, err := NewDBClient(cfg)
	if err != nil {
		t.Fatalf("second NewDBClient failed: %v", err)
	}
	if client2 == nil {
		t.Fatal("second NewDBClient should succeed")
	}
}
