package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestResolveDockerMutantBinary(t *testing.T) {
	oldSourceCodePath := configs.SourceCodePath
	oldTargetPath := configs.TargetPath
	t.Cleanup(func() {
		configs.SourceCodePath = oldSourceCodePath
		configs.TargetPath = oldTargetPath
	})

	root := t.TempDir()
	configs.SourceCodePath = filepath.Join(root, "source_code")
	configs.TargetPath = filepath.Join(configs.SourceCodePath, "target")

	t.Run("falls back to PATH when no local candidate exists", func(t *testing.T) {
		got, err := resolveDockerMutantBinary()
		if err != nil {
			t.Fatalf("resolveDockerMutantBinary() unexpected error: %v", err)
		}
		if got != "malefic-mutant" {
			t.Fatalf("resolveDockerMutantBinary() = %q, want PATH fallback", got)
		}
	})

	t.Run("prefers target release candidate over source bin", func(t *testing.T) {
		releasePath := filepath.Join(configs.TargetPath, "release", "malefic-mutant")
		binPath := filepath.Join(configs.SourceCodePath, "bin", "malefic-mutant")
		writeExecutable(t, releasePath)
		writeExecutable(t, binPath)

		got, err := resolveDockerMutantBinary()
		if err != nil {
			t.Fatalf("resolveDockerMutantBinary() unexpected error: %v", err)
		}
		want := filepath.ToSlash(filepath.Join(ContainerSourceCodePath, "target", "release", "malefic-mutant"))
		if got != want {
			t.Fatalf("resolveDockerMutantBinary() = %q, want %q", got, want)
		}
	})

	t.Run("rejects non executable local candidate", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows does not use Unix execute bits")
		}
		root := t.TempDir()
		configs.SourceCodePath = filepath.Join(root, "source_code")
		configs.TargetPath = filepath.Join(configs.SourceCodePath, "target")

		path := filepath.Join(configs.SourceCodePath, "bin", "malefic-mutant")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("fake mutant"), 0600); err != nil {
			t.Fatalf("write file: %v", err)
		}

		_, err := resolveDockerMutantBinary()
		if err == nil {
			t.Fatal("resolveDockerMutantBinary() expected executable permission error")
		}
		if !strings.Contains(err.Error(), "not executable") {
			t.Fatalf("resolveDockerMutantBinary() error = %q, want contains not executable", err.Error())
		}
	})

	t.Run("rejects directory local candidate", func(t *testing.T) {
		root := t.TempDir()
		configs.SourceCodePath = filepath.Join(root, "source_code")
		configs.TargetPath = filepath.Join(configs.SourceCodePath, "target")

		path := filepath.Join(configs.SourceCodePath, "bin", "malefic-mutant")
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatalf("mkdir candidate: %v", err)
		}

		_, err := resolveDockerMutantBinary()
		if err == nil {
			t.Fatal("resolveDockerMutantBinary() expected directory candidate error")
		}
		if !strings.Contains(err.Error(), "is a directory") {
			t.Fatalf("resolveDockerMutantBinary() error = %q, want contains is a directory", err.Error())
		}
	})
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("fake mutant"), 0700); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}
