package mutant

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func mutantPath() string {
	return BinaryPath()
}

func BinaryPath() string {
	mutantBin := "malefic-mutant"
	if runtime.GOOS == "windows" {
		mutantBin = "malefic-mutant.exe"
	}
	return filepath.Join(configs.BinPath, mutantBin)
}

func CheckBinaryExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("malefic-mutant binary not found: %s", path)
		}
		return fmt.Errorf("failed to stat malefic-mutant binary %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("malefic-mutant binary is a directory: %s", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("malefic-mutant is not executable: %s; run chmod +x %s", path, path)
	}
	return nil
}
