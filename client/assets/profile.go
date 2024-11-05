package assets

import (
	"github.com/chainreactors/logs"
	crConfig "github.com/chainreactors/malice-network/helper/utils/config"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"slices"
)

var (
	maliceProfile = "malice.yaml"

	TargetList = []string{
		"x86_64-apple-darwin",
		"aarch64-apple-darwin",
		"x86_64-unknown-linux-gnu",
		"i686-unknown-linux-gnu",
		"x86_64-pc-windows-msvc",
		"i686-pc-windows-msvc",
		"i686-pc-windows-gnu",
		"x86_64-pc-windows-gnu",
	}

	TypeList = []string{
		"elf",
		"pe",
		"dll",
		"shellcode",
	}
)

type Profile struct {
	ResourceDir string    `yaml:"resources" config:"resources" default:""`
	TempDir     string    `yaml:"tmp" config:"tmp" default:""`
	Aliases     []string  `yaml:"aliases" config:"aliases" default:""`
	Extensions  []string  `yaml:"extensions" config:"extensions" default:""`
	Mals        []string  `yaml:"mals" config:"mals" default:""`
	Settings    *Settings `yaml:"settings" config:"settings"`
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
		logs.Log.Errorf("Failed to find malice.yaml: %v", err)
		os.Exit(0)
		return nil
	}

	profile, err := loadProfile(filePath)
	if err != nil {
		logs.Log.Errorf("Failed to load malice.yaml: %v", err)
		os.Exit(0)
		return nil
	}

	return profile
}

func SaveProfile(profile *Profile) error {
	path, err := findFile(maliceProfile)
	data, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (profile *Profile) AddMal(manifestName string) bool {
	if !slices.Contains(profile.Mals, manifestName) {
		profile.Mals = append(profile.Mals, manifestName)
		return true
	}
	return false
}

func (profile *Profile) AddAlias(alias string) bool {
	if !slices.Contains(profile.Aliases, alias) {
		profile.Aliases = append(profile.Aliases, alias)
		return true
	}
	return false
}

func (profile *Profile) AddExtension(extension string) bool {
	if !slices.Contains(profile.Extensions, extension) {
		profile.Extensions = append(profile.Extensions, extension)
		return true
	}
	return false
}
