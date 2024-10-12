package assets

import (
	"fmt"
	"github.com/chainreactors/logs"
	crConfig "github.com/chainreactors/malice-network/helper/utils/config"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"os"
	"path/filepath"
)

var (
	maliceProfile = "malice.yaml"
)

type Profile struct {
	ResourceDir string   `config:"resources" default:""`
	TempDir     string   `config:"tmp" default:""`
	Aliases     []string `config:"aliases" default:""`
	Extensions  []string `config:"extensions" default:""`
	Mals        []string `config:"mals" default:""`
	//Modules     []string  `yaml:"modules"`
	Settings *Settings `config:"settings"`
}

func findFile(filename string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	workdirPath := filepath.Join(wd, filename)
	if _, err := os.Stat(workdirPath); err == nil {
		return workdirPath, nil
	}

	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execDir := filepath.Dir(execPath)
	execPath = filepath.Join(execDir, filename)
	if _, err := os.Stat(execPath); err == nil {
		return execPath, nil
	}

	if err != nil {
		return "", err
	}
	malicePath := filepath.Join(GetRootAppDir(), filename)
	if _, err := os.Stat(malicePath); err == nil {
		return malicePath, nil
	}

	return malicePath, nil
}

func loadProfile(path string) (*Profile, error) {
	var profile Profile
	config.AddDriver(yaml.Driver)
	if !file.Exist(path) {
		confStr := crConfig.InitDefaultConfig(&profile, 0)
		err := os.WriteFile(path, confStr, 0644)
		if err != nil {
			logs.Log.Errorf("cannot write default config , %s ", err.Error())
			return nil, err
		}
		logs.Log.Warnf("config file not found, created default config %s", path)
	}

	err := crConfig.LoadConfig(path, &profile)
	if err != nil {
		logs.Log.Warnf("cannot load config , %s ", err.Error())
		return nil, err
	}
	return &profile, nil
}

func GetProfile() *Profile {
	filePath, err := findFile(maliceProfile)
	if err != nil {
		logs.Log.Errorf(fmt.Sprintf("Failed to find malice.yaml: %v", err))
		os.Exit(0)
		return nil
	}

	profile, err := loadProfile(filePath)
	if err != nil {
		logs.Log.Errorf(fmt.Sprintf("Failed to load malice.yaml: %v", err))
		os.Exit(0)
		return nil
	}

	return profile
}
