package core

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/spf13/cobra"
)

func TestProcessSearchIndexesAreIsolated(t *testing.T) {
	dir := t.TempDir()
	alive := func(pid int) bool { return pid == 1001 || pid == 1002 }

	first, err := newProcessSearchIndex(dir, 1001, alive)
	if err != nil {
		t.Fatalf("new first process search index: %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })

	second, err := newProcessSearchIndex(dir, 1002, alive)
	if err != nil {
		t.Fatalf("new second process search index: %v", err)
	}
	t.Cleanup(func() { _ = second.Close() })

	if first.path == second.path {
		t.Fatalf("process indexes share path %q", first.path)
	}
	if _, err := os.Stat(first.path); err != nil {
		t.Fatalf("second process removed live first index: %v", err)
	}

	firstCommands := func() []*cobra.Command {
		return []*cobra.Command{{Use: "first-command", Short: "first process command"}}
	}
	secondCommands := func() []*cobra.Command {
		return []*cobra.Command{{Use: "second-command", Short: "second process command"}}
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, rebuild := range []func() error{
		func() error { return first.Rebuild(firstCommands) },
		func() error { return second.Rebuild(secondCommands) },
	} {
		wg.Add(1)
		go func(rebuild func() error) {
			defer wg.Done()
			errs <- rebuild()
		}(rebuild)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent process index rebuild: %v", err)
		}
	}

	results, err := first.Search("second-command", "", "", 10)
	if err != nil {
		t.Fatalf("search first process index: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("first process index contains second process commands: %+v", results)
	}
}

func TestProcessAliveRecognizesCurrentProcess(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatalf("current process %d reported as not alive", os.Getpid())
	}
}

func TestSearchIndexPIDRequiresExactFileName(t *testing.T) {
	valid := []string{"search-123.db", "search-123.db-wal", "search-123.db-shm", "search-123.db-journal"}
	for _, name := range valid {
		if pid, ok := searchIndexPID(name); !ok || pid != 123 {
			t.Errorf("searchIndexPID(%q) = (%d, %v), want (123, true)", name, pid, ok)
		}
	}

	invalid := []string{
		"search.db",
		"search-not-a-pid.db",
		"search-0.db",
		"search-0123.db",
		"search-123.db-backup",
		"search-123.db-wal-journal",
	}
	for _, name := range invalid {
		if pid, ok := searchIndexPID(name); ok {
			t.Errorf("searchIndexPID(%q) = (%d, true), want invalid", name, pid)
		}
	}
}

func TestProcessSearchIndexRemovesStaleFiles(t *testing.T) {
	dir := t.TempDir()
	stalePath := filepath.Join(dir, "search-2001.db")
	livePath := filepath.Join(dir, "search-2002.db")
	unrelatedPath := filepath.Join(dir, "search-not-a-pid.db")

	for _, path := range []string{
		stalePath,
		stalePath + "-wal",
		stalePath + "-shm",
		stalePath + "-journal",
		livePath,
		livePath + "-wal",
		unrelatedPath,
	} {
		if err := os.WriteFile(path, []byte("stale"), 0o600); err != nil {
			t.Fatalf("write fixture %q: %v", path, err)
		}
	}

	alive := func(pid int) bool { return pid == 2002 || pid == 2003 }
	si, err := newProcessSearchIndex(dir, 2003, alive)
	if err != nil {
		t.Fatalf("new process search index: %v", err)
	}
	t.Cleanup(func() { _ = si.Close() })

	for _, path := range searchIndexFiles(stalePath) {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("stale index file still exists %q: %v", path, err)
		}
	}
	for _, path := range []string{livePath, livePath + "-wal", unrelatedPath} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("live or unrelated file was removed %q: %v", path, err)
		}
	}
}

func TestProcessSearchIndexHandlesReusedPID(t *testing.T) {
	dir := t.TempDir()
	pid := 3001
	dbPath := filepath.Join(dir, "search-3001.db")

	if err := os.WriteFile(dbPath, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatalf("write reused PID fixture: %v", err)
	}

	si, err := newProcessSearchIndex(dir, pid, func(candidate int) bool { return candidate == pid })
	if err != nil {
		t.Fatalf("new process search index with reused PID: %v", err)
	}
	t.Cleanup(func() { _ = si.Close() })

	if err := si.Rebuild(func() []*cobra.Command { return []*cobra.Command{{Use: "whoami"}} }); err != nil {
		t.Fatalf("rebuild index after reused PID cleanup: %v", err)
	}
}

func TestProcessSearchIndexCloseRemovesOwnedFiles(t *testing.T) {
	dir := t.TempDir()
	si, err := newProcessSearchIndex(dir, 4001, func(pid int) bool { return pid == 4001 })
	if err != nil {
		t.Fatalf("new process search index: %v", err)
	}

	dbPath := si.path
	if err := os.WriteFile(dbPath+"-journal", []byte("journal"), 0o600); err != nil {
		t.Fatalf("write journal fixture: %v", err)
	}
	if err := si.Close(); err != nil {
		t.Fatalf("close process search index: %v", err)
	}

	for _, path := range searchIndexFiles(dbPath) {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("owned index file still exists %q: %v", path, err)
		}
	}
	if err := si.Close(); err != nil {
		t.Fatalf("second close should be idempotent: %v", err)
	}
}

func TestSearchIndexRebuildReturnsSchemaErrors(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "readonly.db")

	si, err := NewSearchIndex(dbPath)
	if err != nil {
		t.Fatalf("create search index: %v", err)
	}
	if err := si.Close(); err != nil {
		t.Fatalf("close writable search index: %v", err)
	}

	readonlyDB, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro")
	if err != nil {
		t.Fatalf("open readonly search index: %v", err)
	}
	t.Cleanup(func() { _ = readonlyDB.Close() })

	readonlyIndex := &SearchIndex{db: readonlyDB, path: dbPath}
	if err := readonlyIndex.Rebuild(func() []*cobra.Command { return nil }); err == nil {
		t.Fatal("Rebuild succeeded on a readonly database")
	}
}
