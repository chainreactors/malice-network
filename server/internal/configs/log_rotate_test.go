package configs

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotateLogs_RenamesAndCompresses(t *testing.T) {
	dir := t.TempDir()

	// Create a fake log file
	os.WriteFile(filepath.Join(dir, "rpc.log"), []byte("line1\nline2\n"), 0644)
	os.WriteFile(filepath.Join(dir, "auth.log"), []byte("auth data\n"), 0644)

	reopened := false
	RotateLogs(dir, 180, true, func() { reopened = true })

	if !reopened {
		t.Error("reopenFn should have been called")
	}

	today := time.Now().Format("2006-01-02")

	// Original files should be truncated (still exist but empty)
	for _, name := range []string{"rpc.log", "auth.log"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("%s should still exist after rotation: %v", name, err)
		} else if info.Size() != 0 {
			t.Errorf("%s should be truncated to 0 bytes, got %d", name, info.Size())
		}
	}

	// Rotated files should exist with original content
	rotatedRpc := filepath.Join(dir, "rpc."+today+".log")
	rotatedAuth := filepath.Join(dir, "auth."+today+".log")
	if data, err := os.ReadFile(rotatedRpc); err != nil {
		t.Errorf("rotated rpc file should exist: %v", err)
	} else if string(data) != "line1\nline2\n" {
		t.Errorf("rotated rpc file content mismatch: %q", data)
	}
	if data, err := os.ReadFile(rotatedAuth); err != nil {
		t.Errorf("rotated auth file should exist: %v", err)
	} else if string(data) != "auth data\n" {
		t.Errorf("rotated auth file content mismatch: %q", data)
	}
}

func TestRotateLogs_CompressesPreviousDay(t *testing.T) {
	dir := t.TempDir()

	// Create a "yesterday's" rotated log
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	oldFile := filepath.Join(dir, "rpc."+yesterday+".log")
	os.WriteFile(oldFile, []byte("old data\n"), 0644)

	RotateLogs(dir, 180, true, nil)

	// Should be compressed
	if _, err := os.Stat(oldFile); err == nil {
		t.Error("yesterday's .log should have been removed after compression")
	}
	gzFile := oldFile + ".gz"
	if _, err := os.Stat(gzFile); err != nil {
		t.Errorf("compressed file should exist: %v", err)
	}

	// Verify it's valid gzip
	f, _ := os.Open(gzFile)
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("should be valid gzip: %v", err)
	}
	gr.Close()
}

func TestRotateLogs_CleansExpiredFiles(t *testing.T) {
	dir := t.TempDir()

	// Create old rotated files with old mod time
	oldDate := time.Now().AddDate(0, 0, -200).Format("2006-01-02")
	oldLog := filepath.Join(dir, "rpc."+oldDate+".log")
	oldGz := filepath.Join(dir, "auth."+oldDate+".log.gz")
	os.WriteFile(oldLog, []byte("expired"), 0644)
	os.WriteFile(oldGz, []byte("expired gz"), 0644)
	// Set mod time to 200 days ago
	oldTime := time.Now().AddDate(0, 0, -200)
	os.Chtimes(oldLog, oldTime, oldTime)
	os.Chtimes(oldGz, oldTime, oldTime)

	RotateLogs(dir, 180, true, nil)

	if _, err := os.Stat(oldLog); err == nil {
		t.Error("expired .log should have been deleted")
	}
	if _, err := os.Stat(oldGz); err == nil {
		t.Error("expired .log.gz should have been deleted")
	}
}

func TestRotateLogs_NoCompressFlag(t *testing.T) {
	dir := t.TempDir()

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	oldFile := filepath.Join(dir, "rpc."+yesterday+".log")
	os.WriteFile(oldFile, []byte("data\n"), 0644)

	RotateLogs(dir, 180, false, nil)

	// Should NOT be compressed
	if _, err := os.Stat(oldFile); err != nil {
		t.Error("with compress=false, old .log should remain")
	}
	if _, err := os.Stat(oldFile + ".gz"); err == nil {
		t.Error("with compress=false, .gz should not be created")
	}
}

func TestCleanAuditLogs_RemovesOldFiles(t *testing.T) {
	dir := t.TempDir()

	// Recent audit log
	recent := filepath.Join(dir, "abc123.log")
	os.WriteFile(recent, []byte("recent"), 0644)

	// Old audit log
	old := filepath.Join(dir, "def456.log")
	os.WriteFile(old, []byte("old"), 0644)
	oldTime := time.Now().AddDate(0, 0, -200)
	os.Chtimes(old, oldTime, oldTime)

	CleanAuditLogs(dir, 180)

	if _, err := os.Stat(recent); err != nil {
		t.Error("recent audit log should remain")
	}
	if _, err := os.Stat(old); err == nil {
		t.Error("old audit log should be deleted")
	}
}

func TestContainsDate(t *testing.T) {
	tests := []struct {
		name   string
		expect bool
	}{
		{"rpc.2026-03-19.log", true},
		{"auth.2024-01-01.log.gz", true},
		{"rpc.log", false},
		{"auth.log", false},
		{"nodate.txt", false},
	}
	for _, tt := range tests {
		if got := containsDate(tt.name); got != tt.expect {
			t.Errorf("containsDate(%q) = %v, want %v", tt.name, got, tt.expect)
		}
	}
}
