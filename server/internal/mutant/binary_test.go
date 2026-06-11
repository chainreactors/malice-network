package mutant

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestBinaryPathUsesConfiguredBinPath(t *testing.T) {
	oldBinPath := configs.BinPath
	t.Cleanup(func() { configs.BinPath = oldBinPath })

	configs.BinPath = t.TempDir()

	wantName := "malefic-mutant"
	if runtime.GOOS == "windows" {
		wantName = "malefic-mutant.exe"
	}
	if got := BinaryPath(); got != filepath.Join(configs.BinPath, wantName) {
		t.Fatalf("BinaryPath() = %q, want %q", got, filepath.Join(configs.BinPath, wantName))
	}
}

func TestCheckBinaryExecutable(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr string
	}{
		{
			name: "missing file",
			setup: func(t *testing.T) string {
				return filepath.Join(tempDir, "missing-mutant")
			},
			wantErr: "not found",
		},
		{
			name: "directory",
			setup: func(t *testing.T) string {
				path := filepath.Join(tempDir, "mutant-dir")
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				return path
			},
			wantErr: "is a directory",
		},
		{
			name: "non executable file",
			setup: func(t *testing.T) string {
				if runtime.GOOS == "windows" {
					t.Skip("Windows does not use Unix execute bits")
				}
				path := filepath.Join(tempDir, "non-executable")
				if err := os.WriteFile(path, []byte("fake mutant"), 0600); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return path
			},
			wantErr: "not executable",
		},
		{
			name: "executable file",
			setup: func(t *testing.T) string {
				path := filepath.Join(tempDir, "executable")
				perm := os.FileMode(0600)
				if runtime.GOOS != "windows" {
					perm = 0700
				}
				if err := os.WriteFile(path, []byte("fake mutant"), perm); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return path
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			err := CheckBinaryExecutable(path)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("CheckBinaryExecutable(%q) unexpected error: %v", path, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("CheckBinaryExecutable(%q) expected error containing %q", path, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("CheckBinaryExecutable(%q) error = %q, want contains %q", path, err.Error(), tt.wantErr)
			}
		})
	}
}
