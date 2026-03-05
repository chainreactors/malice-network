package configs

import (
	"strings"
	"testing"
)

func TestGetDefaultDatabaseConfig(t *testing.T) {
	cfg := GetDefaultDatabaseConfig()

	if cfg.Dialect != Sqlite {
		t.Errorf("expected dialect %q, got %q", Sqlite, cfg.Dialect)
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("expected MaxIdleConns 10, got %d", cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns != 100 {
		t.Errorf("expected MaxOpenConns 100, got %d", cfg.MaxOpenConns)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("expected LogLevel 'warn', got %q", cfg.LogLevel)
	}
}

func TestDSN_Sqlite(t *testing.T) {
	cfg := &DatabaseConfig{
		Dialect: Sqlite,
	}
	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(dsn, "file:") {
		t.Errorf("SQLite DSN should start with 'file:', got %q", dsn)
	}
	if !strings.Contains(dsn, "malice.db") {
		t.Errorf("SQLite DSN should contain 'malice.db', got %q", dsn)
	}
}

func TestDSN_SqliteWithParams(t *testing.T) {
	cfg := &DatabaseConfig{
		Dialect: Sqlite,
		Params:  map[string]string{"cache": "shared", "_journal_mode": "WAL"},
	}
	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(dsn, "cache=shared") {
		t.Errorf("DSN should contain cache param, got %q", dsn)
	}
	if !strings.Contains(dsn, "_journal_mode=WAL") {
		t.Errorf("DSN should contain journal_mode param, got %q", dsn)
	}
}

func TestDSN_PostgresDefaults(t *testing.T) {
	cfg := &DatabaseConfig{
		Dialect:  Postgres,
		Username: "testuser",
		Password: "testpass",
	}
	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(dsn, "host=localhost") {
		t.Errorf("expected default host=localhost, got %q", dsn)
	}
	if !strings.Contains(dsn, "port=5432") {
		t.Errorf("expected default port=5432, got %q", dsn)
	}
	if !strings.Contains(dsn, "dbname=malice") {
		t.Errorf("expected default dbname=malice, got %q", dsn)
	}
	if !strings.Contains(dsn, "user=testuser") {
		t.Errorf("expected user=testuser, got %q", dsn)
	}
	if !strings.Contains(dsn, "password=testpass") {
		t.Errorf("expected password=testpass, got %q", dsn)
	}
}

func TestDSN_PostgresFullConfig(t *testing.T) {
	cfg := &DatabaseConfig{
		Dialect:  Postgres,
		Host:     "db.example.com",
		Port:     5433,
		Database: "mydb",
		Username: "admin",
		Password: "secret",
		Params:   map[string]string{"sslmode": "require"},
	}
	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(dsn, "host=db.example.com") {
		t.Errorf("expected custom host, got %q", dsn)
	}
	if !strings.Contains(dsn, "port=5433") {
		t.Errorf("expected custom port, got %q", dsn)
	}
	if !strings.Contains(dsn, "dbname=mydb") {
		t.Errorf("expected custom dbname, got %q", dsn)
	}
	if !strings.Contains(dsn, "sslmode=require") {
		t.Errorf("expected sslmode param, got %q", dsn)
	}
}

func TestDSN_InvalidDialect(t *testing.T) {
	cfg := &DatabaseConfig{
		Dialect: "oracle",
	}
	_, err := cfg.DSN()
	if err == nil {
		t.Fatal("expected error for invalid dialect, got nil")
	}
	if err != ErrInvalidDialect {
		t.Errorf("expected ErrInvalidDialect, got %v", err)
	}
}

func TestDSN_EmptyDialect(t *testing.T) {
	cfg := &DatabaseConfig{
		Dialect: "",
	}
	_, err := cfg.DSN()
	if err == nil {
		t.Fatal("expected error for empty dialect, got nil")
	}
	if err != ErrInvalidDialect {
		t.Errorf("expected ErrInvalidDialect, got %v", err)
	}
}

func TestEncodeParamsPostgres_Empty(t *testing.T) {
	result := encodeParamsPostgres(nil)
	if result != "" {
		t.Errorf("expected empty string for nil params, got %q", result)
	}
}

func TestEncodeParamsPostgres_Single(t *testing.T) {
	result := encodeParamsPostgres(map[string]string{"sslmode": "disable"})
	if result != "sslmode=disable" {
		t.Errorf("expected 'sslmode=disable', got %q", result)
	}
}

func TestEncodeParamsPostgres_Multiple(t *testing.T) {
	params := map[string]string{
		"sslmode":         "require",
		"connect_timeout": "10",
	}
	result := encodeParamsPostgres(params)
	if !strings.Contains(result, "sslmode=require") {
		t.Errorf("missing sslmode in %q", result)
	}
	if !strings.Contains(result, "connect_timeout=10") {
		t.Errorf("missing connect_timeout in %q", result)
	}
	// Should be space-separated
	parts := strings.Split(result, " ")
	if len(parts) != 2 {
		t.Errorf("expected 2 space-separated parts, got %d: %q", len(parts), result)
	}
}

func TestDialectConstants(t *testing.T) {
	if Sqlite != "sqlite3" {
		t.Errorf("Sqlite constant: expected 'sqlite3', got %q", Sqlite)
	}
	if Postgres != "postgresql" {
		t.Errorf("Postgres constant: expected 'postgresql', got %q", Postgres)
	}
	if MySQL != "mysql" {
		t.Errorf("MySQL constant: expected 'mysql', got %q", MySQL)
	}
}
