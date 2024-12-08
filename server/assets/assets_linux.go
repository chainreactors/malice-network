package assets

import (
	"embed"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

var (
	//go:embed  linux/*
	assetsFs embed.FS
)

func SetupGithubFile() error {
	mutant, err := assetsFs.ReadFile("linux/malefic-mutant")
	if err != nil {
		logs.Log.Errorf("malefic-mutant asset not found")
	}

	sgn, err := assetsFs.ReadFile("linux/sgn")
	if err != nil {
		logs.Log.Errorf("sgn asset not found")
	}

	err = os.WriteFile(filepath.Join(configs.BinPath, "malefic-mutant"), mutant, 0600)
	if err != nil {
		logs.Log.Errorf("Failed to write malefic-mutant data %s to: by %s", configs.BinPath, err)
	}

	err = os.WriteFile(filepath.Join(configs.BinPath, "sgn"), sgn, 0600)
	if err != nil {
		logs.Log.Errorf("Failed to write sgn data %s to: by %s", configs.BinPath, err)
	}
	return nil
}
