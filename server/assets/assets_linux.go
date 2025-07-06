package assets

import (
	"embed"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"os"
	"path/filepath"
)

var (
	//go:embed  linux/*
	assetsFs embed.FS
)

func SetupGithubFile() error {
	sgn, err := assetsFs.ReadFile("linux/sgn")
	if err != nil {
		logs.Log.Errorf("sgn asset not found")
	}

	err = os.WriteFile(filepath.Join(configs.BinPath, "sgn"), sgn, 0700)
	if err != nil {
		logs.Log.Errorf("Failed to write sgn data %s to: by %s", configs.BinPath, err)
	}
	return nil
}
