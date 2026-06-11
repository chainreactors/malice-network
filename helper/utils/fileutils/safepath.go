package fileutils

import (
	"errors"
	"path"
	"path/filepath"
	"strings"
)

var ErrUnsafePath = errors.New("unsafe path")

// SafeJoin joins baseDir and unsafeRelPath and ensures the final path stays within baseDir.
// It rejects absolute paths and parent directory traversal ("..").
func SafeJoin(baseDir string, unsafeRelPath string) (string, error) {
	baseDir = filepath.Clean(baseDir)
	if baseDir == "" || baseDir == "." {
		return "", errors.New("base directory is empty")
	}
	if unsafeRelPath == "" {
		return "", errors.New("path is empty")
	}

	rel := filepath.Clean(unsafeRelPath)
	if rel == "" || rel == "." {
		return "", ErrUnsafePath
	}
	if filepath.IsAbs(rel) {
		return "", ErrUnsafePath
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrUnsafePath
	}

	full := filepath.Clean(filepath.Join(baseDir, rel))
	if full != baseDir && !strings.HasPrefix(full, baseDir+string(filepath.Separator)) {
		return "", ErrUnsafePath
	}
	return full, nil
}

// SanitizeBasename extracts and validates a single safe filename component from an untrusted path-like input.
// It normalizes Windows separators and rejects empty, "." and "..".
func SanitizeBasename(unsafePath string) (string, error) {
	unsafePath = strings.TrimSpace(unsafePath)
	if unsafePath == "" {
		return "", errors.New("filename is empty")
	}
	normalized := strings.ReplaceAll(unsafePath, "\\", "/")
	base := path.Base(normalized)
	base = strings.TrimSpace(base)
	switch base {
	case "", ".", "..", "/":
		return "", ErrUnsafePath
	}
	if strings.Contains(base, "/") || strings.Contains(base, "\\") {
		return "", ErrUnsafePath
	}
	return base, nil
}
