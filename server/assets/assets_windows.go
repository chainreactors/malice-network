package assets

import (
	"embed"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"os"
	"path/filepath"
)

var (
	//go:embed windows/*
	assetsFs embed.FS
)

func SetupGithubFile() error {
	mutant, err := assetsFs.ReadFile("windows/malefic-mutant.exe")
	if err != nil {
		logs.Log.Errorf("malefic-mutant.exe asset not found")
		return err
	}

	//linuxMutant, err := assetsFs.ReadFile("windows/malefic-mutant")
	//if err != nil {
	//	logs.Log.Errorf("malefic-mutant asset not found")
	//	return err
	//}

	sgn, err := assetsFs.ReadFile("windows/sgn.exe")
	if err != nil {
		logs.Log.Errorf("sgn asset not found")
	}

	dll, err := assetsFs.ReadFile("windows/keystone.dll")
	if err != nil {
		logs.Log.Errorf("keystone.dll asset not found")
	}

	err = os.WriteFile(filepath.Join(configs.BinPath, "malefic-mutant.exe"), mutant, 0700)
	if err != nil {
		logs.Log.Errorf("Failed to write malefic-mutant data to: %s by %s", configs.BinPath, err)
	}

	//err = os.WriteFile(filepath.Join(configs.BinPath, "malefic-mutant"), linuxMutant, 0600)
	//if err != nil {
	//	logs.Log.Errorf("Failed to write malefic-mutant data to: %s by %s", configs.BinPath, err)
	//	return err
	//}

	err = os.WriteFile(filepath.Join(configs.BinPath, "sgn.exe"), sgn, 0700)
	if err != nil {
		logs.Log.Errorf("Failed to write sgn data to: %s by %s", configs.BinPath, err)
	}

	err = os.WriteFile(filepath.Join(configs.BinPath, "keystone.dll"), dll, 0700)
	if err != nil {
		logs.Log.Errorf("Failed to write keystone.dll data %s to: by %s", configs.BinPath, err)
	}
	return nil
}
