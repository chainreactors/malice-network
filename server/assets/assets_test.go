package assets

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestSetupGithubFileCreatesBinPath(t *testing.T) {
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))

	if err := SetupGithubFile(); err != nil {
		t.Fatalf("SetupGithubFile() error = %v", err)
	}

	names := []string{"sgn", "malefic-mutant"}
	if runtime.GOOS == "windows" {
		names = []string{"sgn.exe", "malefic-mutant.exe"}
	}

	for _, name := range names {
		path := filepath.Join(configs.BinPath, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("expected %s to be non-empty", path)
		}
	}
}
