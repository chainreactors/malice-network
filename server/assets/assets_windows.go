package assets

import (
	"embed"
	"fmt"
	"runtime"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"os"
	"path/filepath"
)

var (
	//go:embed windows/amd64/* windows/arm64/*
	assetsFs embed.FS
)

func SetupGithubFile() error {
	arch := runtime.GOARCH

	// Read sgn (available for both amd64 and arm64)
	sgnPath := fmt.Sprintf("windows/%s/sgn.exe", arch)
	sgn, err := assetsFs.ReadFile(sgnPath)
	if err != nil {
		logs.Log.Errorf("sgn.exe asset not found at %s: %v", sgnPath, err)
		return err
	}

	// Read malefic-mutant (only amd64 available, fallback for arm64)
	mutantPath := fmt.Sprintf("windows/%s/malefic-mutant.exe", arch)
	mutant, err := assetsFs.ReadFile(mutantPath)
	if err != nil && arch == "arm64" {
		// Fallback to amd64 for arm64 (Windows arm64 has x64 emulation)
		logs.Log.Warnf("malefic-mutant.exe not found for arm64, falling back to amd64")
		mutantPath = "windows/amd64/malefic-mutant.exe"
		mutant, err = assetsFs.ReadFile(mutantPath)
	}
	if err != nil {
		logs.Log.Errorf("malefic-mutant.exe asset not found: %v", err)
	}

	if err = os.MkdirAll(configs.BinPath, 0700); err != nil {
		logs.Log.Errorf("Failed to create bin path %s: %v", configs.BinPath, err)
		return err
	}

	err = os.WriteFile(filepath.Join(configs.BinPath, "sgn.exe"), sgn, 0700)
	if err != nil {
		logs.Log.Errorf("Failed to write sgn.exe data to %s: %v", configs.BinPath, err)
		return err
	}

	if mutant != nil {
		err = os.WriteFile(filepath.Join(configs.BinPath, "malefic-mutant.exe"), mutant, 0700)
		if err != nil {
			logs.Log.Errorf("Failed to write malefic-mutant.exe data to %s: %v", configs.BinPath, err)
			return err
		}
	}

	return nil
}
