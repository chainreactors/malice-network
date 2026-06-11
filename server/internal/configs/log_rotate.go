package configs

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chainreactors/logs"
)

const (
	DefaultLogMaxAge   = 180  // days
	DefaultLogCompress = true // gzip old logs
)

// RotateLogs rotates .log files in logDir:
//  1. Rename current .log → .{date}.log
//  2. Compress yesterday's .{date}.log → .{date}.log.gz (if compress=true)
//  3. Delete rotated logs older than maxAge days
func RotateLogs(logDir string, maxAge int, compress bool, reopenFn func()) {
	if maxAge <= 0 {
		maxAge = DefaultLogMaxAge
	}
	today := time.Now().Format("2006-01-02")
	cutoff := time.Now().AddDate(0, 0, -maxAge)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		logs.Log.Errorf("[log-rotate] failed to read log dir %s: %v", logDir, err)
		return
	}

	// Step 1: rotate active .log files (copy + truncate to avoid Windows locked-file errors)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".log") || containsDate(name) {
			continue
		}
		// e.g. rpc.log → rpc.2026-03-19.log
		base := strings.TrimSuffix(name, ".log")
		rotated := base + "." + today + ".log"
		src := filepath.Join(logDir, name)
		dst := filepath.Join(logDir, rotated)
		if err := copyAndTruncate(src, dst); err != nil {
			logs.Log.Errorf("[log-rotate] failed to rotate %s → %s: %v", name, rotated, err)
		}
	}

	// Step 2: reopen loggers so they write to fresh files
	if reopenFn != nil {
		reopenFn()
	}

	// Re-read after rotation
	entries, err = os.ReadDir(logDir)
	if err != nil {
		return
	}

	// Step 3: compress old rotated logs and clean expired
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		fullPath := filepath.Join(logDir, name)

		// Clean expired .log.gz and old rotated .log
		if strings.HasSuffix(name, ".log.gz") || (strings.HasSuffix(name, ".log") && containsDate(name)) {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(fullPath)
				continue
			}
		}

		// Compress rotated .log files (not today's)
		if compress && strings.HasSuffix(name, ".log") && containsDate(name) && !strings.Contains(name, today) {
			if err := gzipFile(fullPath); err != nil {
				logs.Log.Errorf("[log-rotate] failed to compress %s: %v", name, err)
			}
		}
	}
}

// CleanAuditLogs removes audit log files older than maxAge days.
func CleanAuditLogs(auditDir string, maxAge int) {
	if maxAge <= 0 {
		maxAge = DefaultLogMaxAge
	}
	cutoff := time.Now().AddDate(0, 0, -maxAge)

	entries, err := os.ReadDir(auditDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(auditDir, entry.Name()))
		}
	}
}

// containsDate checks if a filename contains a date pattern like .2006-01-02.
func containsDate(name string) bool {
	// Look for .YYYY-MM-DD. pattern
	for i := 0; i+10 <= len(name); i++ {
		if name[i] == '.' && i+11 <= len(name) &&
			name[i+5] == '-' && name[i+8] == '-' {
			_, err := time.Parse("2006-01-02", name[i+1:i+11])
			if err == nil {
				return true
			}
		}
	}
	return false
}

// copyAndTruncate copies src to dst and truncates src to zero bytes.
// Unlike os.Rename, this works on Windows even when src is held open by another process.
func copyAndTruncate(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	in.Close()

	return os.Truncate(src, 0)
}

// gzipFile compresses src to src.gz and removes the original.
func gzipFile(src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(src + ".gz")
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	gz.Name = filepath.Base(src)
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		os.Remove(src + ".gz")
		return err
	}
	if err := gz.Close(); err != nil {
		os.Remove(src + ".gz")
		return err
	}

	in.Close()
	return os.Remove(src)
}
