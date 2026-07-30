package db

import (
	"strings"
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
	&models.SessionLink{},
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
		"profiles", "website_contents", "sessions", "session_links", "artifacts", "tasks", "contexts"} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("table %q should exist after AutoMigrate", table)
		}
	}
}

func TestNewDBClient_SQLiteSchemaHasNoInvalidForeignKeys(t *testing.T) {
	setupTestDB(t)

	client, err := NewDBClient(&configs.DatabaseConfig{Dialect: configs.Sqlite})
	if err != nil {
		t.Fatalf("NewDBClient failed: %v", err)
	}

	assertSQLiteForeignKeyCheckOK(t, client)
	assertSQLiteTableSQLNotContains(t, client, "sessions", "fk_tasks_session", "fk_contexts_session")
	assertSQLiteTableSQLNotContains(t, client, "contexts", "fk_contexts_pipeline")
	assertSQLiteTableSQLNotContains(t, client, "website_contents", "fk_website_contents_pipeline")
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

func TestNewDBClient_DropsLegacyPipelineNameUniqueIndex(t *testing.T) {
	setupTestDB(t)

	cfg := &configs.DatabaseConfig{Dialect: configs.Sqlite}
	client, err := NewDBClient(cfg)
	if err != nil {
		t.Fatalf("NewDBClient failed: %v", err)
	}
	if err := client.Exec("CREATE UNIQUE INDEX idx_pipelines_name ON pipelines(name)").Error; err != nil {
		t.Fatalf("failed to create legacy unique index: %v", err)
	}

	client, err = NewDBClient(cfg)
	if err != nil {
		t.Fatalf("second NewDBClient failed: %v", err)
	}
	if client.Migrator().HasIndex(&models.Pipeline{}, "idx_pipelines_name") {
		t.Fatal("legacy pipeline name index should be removed")
	}

	oldClient := Client
	t.Cleanup(func() {
		Client = oldClient
	})
	Client = client
	if _, err := SavePipeline(newTestPipeline("legacy-shared", "ls-a")); err != nil {
		t.Fatalf("SavePipeline listener A failed: %v", err)
	}
	if _, err := SavePipeline(newTestPipeline("legacy-shared", "ls-b")); err != nil {
		t.Fatalf("SavePipeline listener B failed after legacy index cleanup: %v", err)
	}
}

func TestNewDBClient_CleansLegacySQLiteConstraints(t *testing.T) {
	setupTestDB(t)

	legacy, err := sqliteClient(&configs.DatabaseConfig{Dialect: configs.Sqlite})
	if err != nil {
		t.Fatalf("open legacy sqlite db: %v", err)
	}
	createLegacySQLiteConstraintSchema(t, legacy)
	if sqlDB, err := legacy.DB(); err == nil {
		_ = sqlDB.Close()
	}

	client, err := NewDBClient(&configs.DatabaseConfig{Dialect: configs.Sqlite})
	if err != nil {
		t.Fatalf("NewDBClient failed: %v", err)
	}

	assertSQLiteForeignKeyCheckOK(t, client)
	assertSQLiteTableSQLNotContains(t, client, "sessions", "fk_tasks_session", "fk_contexts_session")
	assertSQLiteTableSQLNotContains(t, client, "contexts", "fk_contexts_pipeline")
	assertSQLiteTableSQLNotContains(t, client, "website_contents", "fk_website_contents_pipeline")
	assertSQLiteTableSQLNotContains(t, client, "pipelines", "uni_pipelines_name", "UNIQUE (`name`)")

	oldClient := Client
	t.Cleanup(func() {
		Client = oldClient
	})
	Client = client
	if _, err := SavePipeline(newTestPipeline("legacy-shared", "ls-a")); err != nil {
		t.Fatalf("SavePipeline listener A failed: %v", err)
	}
	if _, err := SavePipeline(newTestPipeline("legacy-shared", "ls-b")); err != nil {
		t.Fatalf("SavePipeline listener B failed after legacy constraint cleanup: %v", err)
	}
}

func createLegacySQLiteConstraintSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	statements := []string{
		`CREATE TABLE pipelines (
			id uuid,
			created_at datetime,
			listener_id text,
			name text,
			ip text DEFAULT "",
			host text,
			port integer,
			type text,
			enable boolean,
			params text,
			cert_name text,
			PRIMARY KEY (id),
			CONSTRAINT uni_pipelines_name UNIQUE (name)
		)`,
		`CREATE UNIQUE INDEX idx_pipelines_listener_name ON pipelines(listener_id, name)`,
		`CREATE TABLE tasks (
			id text,
			created datetime,
			deadline datetime,
			call_by text,
			seq integer,
			type text,
			session_id text,
			cur integer,
			total integer,
			description text,
			client_name text,
			finish_time datetime,
			last_time datetime,
			PRIMARY KEY (id)
		)`,
		`CREATE TABLE profiles (
			id uuid,
			name text,
			params text,
			pipeline_id text,
			source text DEFAULT "user",
			source_hash text DEFAULT "",
			created_at datetime,
			deleted_at datetime,
			listener_id text,
			PRIMARY KEY (id),
			CONSTRAINT uni_profiles_name UNIQUE (name)
		)`,
		`CREATE TABLE contexts (
			id uuid,
			created_at datetime,
			updated_at datetime,
			session_id text,
			pipeline_id text,
			task_id text,
			type text,
			nonce text,
			value blob,
			PRIMARY KEY (id),
			CONSTRAINT fk_contexts_pipeline FOREIGN KEY (pipeline_id) REFERENCES pipelines(name) ON DELETE SET NULL ON UPDATE CASCADE,
			CONSTRAINT fk_contexts_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL ON UPDATE CASCADE
		)`,
		`CREATE TABLE website_contents (
			id uuid,
			created_at datetime,
			file text,
			path text,
			size integer,
			type text,
			content_type text,
			auth text,
			pipeline_id text,
			listener_id text,
			PRIMARY KEY (id),
			CONSTRAINT fk_website_contents_pipeline FOREIGN KEY (pipeline_id) REFERENCES pipelines(name) ON DELETE SET NULL ON UPDATE CASCADE
		)`,
		`CREATE TABLE sessions (
			session_id text,
			raw_id integer,
			created_at datetime,
			note text,
			group_name text,
			target text,
			initialized numeric,
			type text,
			pipeline_id text,
			listener_id text,
			is_alive numeric,
			last_checkin integer,
			is_removed numeric DEFAULT false,
			data text,
			profile_name text,
			PRIMARY KEY (session_id),
			CONSTRAINT fk_contexts_session FOREIGN KEY (session_id) REFERENCES contexts(session_id) ON DELETE SET NULL ON UPDATE CASCADE,
			CONSTRAINT fk_sessions_profile FOREIGN KEY (profile_name) REFERENCES profiles(name) ON DELETE SET NULL ON UPDATE CASCADE,
			CONSTRAINT fk_tasks_session FOREIGN KEY (session_id) REFERENCES tasks(session_id) ON DELETE SET NULL ON UPDATE CASCADE
		)`,
		`INSERT INTO pipelines (id, listener_id, name, type) VALUES ('pipe-old', 'ls-old', 'legacy-shared', 'tcp')`,
		`INSERT INTO contexts (id, session_id, pipeline_id, task_id, type) VALUES ('ctx-old', 'sess-old', 'legacy-shared', 'sess-old-1', 'task')`,
		`INSERT INTO website_contents (id, path, pipeline_id, listener_id) VALUES ('web-old', '/old', 'legacy-shared', 'ls-old')`,
		`INSERT INTO sessions (session_id, listener_id, pipeline_id) VALUES ('sess-old', 'ls-old', 'legacy-shared')`,
		`INSERT INTO tasks (id, session_id, seq, type) VALUES ('sess-old-1', 'sess-old', 1, 'ping')`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("legacy schema statement failed: %v\n%s", err, stmt)
		}
	}
}

func assertSQLiteForeignKeyCheckOK(t *testing.T, db *gorm.DB) {
	t.Helper()

	rows, err := db.Raw("PRAGMA foreign_key_check").Rows()
	if err != nil {
		t.Fatalf("PRAGMA foreign_key_check failed: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		var table string
		var rowID int64
		var parent string
		var fkID int64
		if err := rows.Scan(&table, &rowID, &parent, &fkID); err != nil {
			t.Fatalf("scan foreign_key_check row: %v", err)
		}
		t.Fatalf("foreign key violation: table=%s rowid=%d parent=%s fkid=%d", table, rowID, parent, fkID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read foreign_key_check rows: %v", err)
	}
}

func assertSQLiteTableSQLNotContains(t *testing.T, db *gorm.DB, table string, markers ...string) {
	t.Helper()

	var ddl string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table).Row().Scan(&ddl); err != nil {
		t.Fatalf("read sqlite schema for %s: %v", table, err)
	}
	for _, marker := range markers {
		if strings.Contains(ddl, marker) {
			t.Fatalf("table %s still contains legacy schema marker %q in DDL: %s", table, marker, ddl)
		}
	}
}
