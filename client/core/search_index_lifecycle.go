package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const searchIndexDirPerm = 0o700

type processAliveFunc func(pid int) bool

func newCurrentProcessSearchIndex(dir string) (*SearchIndex, error) {
	return newProcessSearchIndex(dir, os.Getpid(), processAlive)
}

func newProcessSearchIndex(dir string, pid int, alive processAliveFunc) (*SearchIndex, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid search index process ID %d", pid)
	}
	if alive == nil {
		return nil, errors.New("process liveness check is required")
	}
	if err := os.MkdirAll(dir, searchIndexDirPerm); err != nil {
		return nil, fmt.Errorf("create search index directory: %w", err)
	}

	dbPath := filepath.Join(dir, fmt.Sprintf("search-%d.db", pid))
	if err := removeSearchIndexFiles(dbPath); err != nil {
		return nil, fmt.Errorf("remove reused search index for process %d: %w", pid, err)
	}
	if err := removeStaleSearchIndexes(dir, pid, alive); err != nil {
		return nil, fmt.Errorf("remove stale search indexes: %w", err)
	}

	si, err := newSearchIndex(dbPath, true)
	if err != nil {
		return nil, err
	}
	return si, nil
}

func removeStaleSearchIndexes(dir string, currentPID int, alive processAliveFunc) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	stale := make(map[int]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		pid, ok := searchIndexPID(entry.Name())
		if !ok || pid == currentPID || alive(pid) {
			continue
		}
		stale[pid] = filepath.Join(dir, fmt.Sprintf("search-%d.db", pid))
	}

	var cleanupErr error
	for pid, dbPath := range stale {
		if err := removeSearchIndexFiles(dbPath); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("process %d: %w", pid, err))
		}
	}
	return cleanupErr
}

func searchIndexPID(name string) (int, bool) {
	var base string
	switch {
	case strings.HasSuffix(name, ".db"):
		base = name
	case strings.HasSuffix(name, ".db-wal"):
		base = strings.TrimSuffix(name, "-wal")
	case strings.HasSuffix(name, ".db-shm"):
		base = strings.TrimSuffix(name, "-shm")
	case strings.HasSuffix(name, ".db-journal"):
		base = strings.TrimSuffix(name, "-journal")
	default:
		return 0, false
	}
	if !strings.HasPrefix(base, "search-") || !strings.HasSuffix(base, ".db") {
		return 0, false
	}

	pidText := strings.TrimSuffix(strings.TrimPrefix(base, "search-"), ".db")
	pid, err := strconv.Atoi(pidText)
	if err != nil || pid <= 0 || strconv.Itoa(pid) != pidText {
		return 0, false
	}
	return pid, true
}

func searchIndexFiles(dbPath string) []string {
	return []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"}
}

func removeSearchIndexFiles(dbPath string) error {
	var cleanupErr error
	for _, path := range searchIndexFiles(dbPath) {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove %q: %w", path, err))
		}
	}
	return cleanupErr
}
