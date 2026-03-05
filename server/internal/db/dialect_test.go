package db

import (
	"strings"
	"testing"
)

func TestSqliteAdapterName(t *testing.T) {
	a := &sqliteAdapter{}
	if a.Name() != "sqlite" {
		t.Errorf("expected 'sqlite', got %q", a.Name())
	}
}

func TestPostgresAdapterName(t *testing.T) {
	a := &postgresAdapter{}
	if a.Name() != "postgres" {
		t.Errorf("expected 'postgres', got %q", a.Name())
	}
}

func TestSqliteAdapterFindAliveSessionsUpdateSQL(t *testing.T) {
	a := &sqliteAdapter{}
	sql := a.FindAliveSessionsUpdateSQL()

	mustContain := []string{
		"UPDATE sessions",
		"SET is_alive = false",
		"strftime('%s', 'now')",
		"JSON_EXTRACT(data, '$.interval')",
		"is_removed = false",
	}
	for _, s := range mustContain {
		if !strings.Contains(sql, s) {
			t.Errorf("SQLite update SQL missing %q", s)
		}
	}
}

func TestSqliteAdapterFindAliveSessionsSelectSQL(t *testing.T) {
	a := &sqliteAdapter{}
	sql := a.FindAliveSessionsSelectSQL()

	mustContain := []string{
		"SELECT *",
		"FROM sessions",
		"strftime('%s', 'now')",
		"JSON_EXTRACT(data, '$.interval')",
		"is_removed = false",
	}
	for _, s := range mustContain {
		if !strings.Contains(sql, s) {
			t.Errorf("SQLite select SQL missing %q", s)
		}
	}
}

func TestPostgresAdapterFindAliveSessionsUpdateSQL(t *testing.T) {
	a := &postgresAdapter{}
	sql := a.FindAliveSessionsUpdateSQL()

	mustContain := []string{
		"UPDATE sessions",
		"SET is_alive = false",
		"EXTRACT(EPOCH FROM NOW())",
		"data::json->>'interval'",
		"is_removed = false",
	}
	for _, s := range mustContain {
		if !strings.Contains(sql, s) {
			t.Errorf("Postgres update SQL missing %q", s)
		}
	}
}

func TestPostgresAdapterFindAliveSessionsSelectSQL(t *testing.T) {
	a := &postgresAdapter{}
	sql := a.FindAliveSessionsSelectSQL()

	mustContain := []string{
		"SELECT *",
		"FROM sessions",
		"EXTRACT(EPOCH FROM NOW())",
		"data::json->>'interval'",
		"is_removed = false",
	}
	for _, s := range mustContain {
		if !strings.Contains(sql, s) {
			t.Errorf("Postgres select SQL missing %q", s)
		}
	}
}

func TestPostgresAdapterEmptyDataProtection(t *testing.T) {
	a := &postgresAdapter{}

	for _, sql := range []string{
		a.FindAliveSessionsUpdateSQL(),
		a.FindAliveSessionsSelectSQL(),
	} {
		if !strings.Contains(sql, "CASE WHEN data IS NOT NULL AND data != ''") {
			t.Errorf("Postgres SQL missing empty data protection: %s", sql)
		}
	}
}

func TestSqliteAdapterAppendLogExpr(t *testing.T) {
	a := &sqliteAdapter{}
	expr := a.AppendLogExpr("test log")

	if expr.SQL != "ifnull(log, '') || ?" {
		t.Errorf("unexpected SQL: %q", expr.SQL)
	}
	if len(expr.Vars) != 1 || expr.Vars[0] != "test log" {
		t.Errorf("unexpected vars: %v", expr.Vars)
	}
}

func TestPostgresAdapterAppendLogExpr(t *testing.T) {
	a := &postgresAdapter{}
	expr := a.AppendLogExpr("test log")

	if expr.SQL != "COALESCE(log, '') || ?" {
		t.Errorf("unexpected SQL: %q", expr.SQL)
	}
	if len(expr.Vars) != 1 || expr.Vars[0] != "test log" {
		t.Errorf("unexpected vars: %v", expr.Vars)
	}
}

func TestSqliteAdapterCastIDAsText(t *testing.T) {
	a := &sqliteAdapter{}
	result := a.CastIDAsText("id")
	expected := "CAST(id AS TEXT) LIKE ?"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPostgresAdapterCastIDAsText(t *testing.T) {
	a := &postgresAdapter{}
	result := a.CastIDAsText("id")
	expected := "id::text LIKE ?"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSqliteAdapterDateFunction(t *testing.T) {
	a := &sqliteAdapter{}
	result := a.DateFunction("contexts.created_at")
	expected := "DATE(contexts.created_at)"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPostgresAdapterDateFunction(t *testing.T) {
	a := &postgresAdapter{}
	result := a.DateFunction("contexts.created_at")
	expected := "contexts.created_at::date"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDialectAdapterInterface(t *testing.T) {
	var _ DialectAdapter = &sqliteAdapter{}
	var _ DialectAdapter = &postgresAdapter{}
}

func TestAdapterCastIDAsTextWithDifferentColumns(t *testing.T) {
	tests := []struct {
		adapter  DialectAdapter
		column   string
		expected string
	}{
		{&sqliteAdapter{}, "id", "CAST(id AS TEXT) LIKE ?"},
		{&sqliteAdapter{}, "session_id", "CAST(session_id AS TEXT) LIKE ?"},
		{&postgresAdapter{}, "id", "id::text LIKE ?"},
		{&postgresAdapter{}, "session_id", "session_id::text LIKE ?"},
	}

	for _, tt := range tests {
		result := tt.adapter.CastIDAsText(tt.column)
		if result != tt.expected {
			t.Errorf("%s.CastIDAsText(%q): expected %q, got %q",
				tt.adapter.Name(), tt.column, tt.expected, result)
		}
	}
}

func TestAdapterDateFunctionWithDifferentColumns(t *testing.T) {
	tests := []struct {
		adapter  DialectAdapter
		column   string
		expected string
	}{
		{&sqliteAdapter{}, "created_at", "DATE(created_at)"},
		{&sqliteAdapter{}, "contexts.created_at", "DATE(contexts.created_at)"},
		{&postgresAdapter{}, "created_at", "created_at::date"},
		{&postgresAdapter{}, "contexts.created_at", "contexts.created_at::date"},
	}

	for _, tt := range tests {
		result := tt.adapter.DateFunction(tt.column)
		if result != tt.expected {
			t.Errorf("%s.DateFunction(%q): expected %q, got %q",
				tt.adapter.Name(), tt.column, tt.expected, result)
		}
	}
}

func TestSqliteNoEmptyDataProtection(t *testing.T) {
	a := &sqliteAdapter{}
	updateSQL := a.FindAliveSessionsUpdateSQL()
	selectSQL := a.FindAliveSessionsSelectSQL()

	// SQLite's JSON_EXTRACT returns NULL for invalid JSON, so no CASE WHEN needed
	if strings.Contains(updateSQL, "CASE WHEN") {
		t.Error("SQLite update SQL should not contain CASE WHEN protection")
	}
	if strings.Contains(selectSQL, "CASE WHEN") {
		t.Error("SQLite select SQL should not contain CASE WHEN protection")
	}
}
