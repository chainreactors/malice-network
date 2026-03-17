package fileutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicWriteFileRejectsEmptyPath(t *testing.T) {
	err := AtomicWriteFile("", []byte("data"), 0o600)
	if err == nil || !strings.Contains(err.Error(), "path is empty") {
		t.Fatalf("AtomicWriteFile(empty) error = %v, want empty path error", err)
	}
}

func TestAtomicWriteFileOverwritesTargetContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "file.txt")
	if err := AtomicWriteFile(path, []byte("first"), 0o600); err != nil {
		t.Fatalf("AtomicWriteFile(first) failed: %v", err)
	}
	if err := AtomicWriteFile(path, []byte("second"), 0o600); err != nil {
		t.Fatalf("AtomicWriteFile(second) failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "second" {
		t.Fatalf("content = %q, want second", string(content))
	}
}
