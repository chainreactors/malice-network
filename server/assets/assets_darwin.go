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
	//go:embed darwin/amd64/* darwin/arm64/*
	assetsFs embed.FS
)

func SetupGithubFile() error {
	arch := runtime.GOARCH

	// Read sgn (available for both amd64 and arm64)
	sgnPath := fmt.Sprintf("darwin/%s/sgn", arch)
	sgn, err := assetsFs.ReadFile(sgnPath)
	if err != nil {
		logs.Log.Errorf("sgn asset not found at %s: %v", sgnPath, err)
		return err
	}

	// Read malefic-mutant (available for both amd64 and arm64)
	mutantPath := fmt.Sprintf("darwin/%s/malefic-mutant", arch)
	mutant, err := assetsFs.ReadFile(mutantPath)
	if err != nil {
		logs.Log.Errorf("malefic-mutant asset not found at %s: %v", mutantPath, err)
	}

	if err = os.MkdirAll(configs.BinPath, 0700); err != nil {
		logs.Log.Errorf("Failed to create bin path %s: %v", configs.BinPath, err)
		return err
	}

	err = os.WriteFile(filepath.Join(configs.BinPath, "sgn"), sgn, 0700)
	if err != nil {
		logs.Log.Errorf("Failed to write sgn data to %s: %v", configs.BinPath, err)
		return err
	}

	if mutant != nil {
		err = os.WriteFile(filepath.Join(configs.BinPath, "malefic-mutant"), mutant, 0700)
		if err != nil {
			logs.Log.Errorf("Failed to write malefic-mutant data to %s: %v", configs.BinPath, err)
			return err
		}
	}

	return nil
}
