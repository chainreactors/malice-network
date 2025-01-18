package assets

import (
	"github.com/chainreactors/logs"
	"github.com/gookit/config/v2"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var (
	maliceProfile = "malice.yaml"
)

var HookFn = func(event string, c *config.Config) {
	p := &Profile{}
	if event == config.OnSetValue {
		err := c.MapStruct("", p)
		if err != nil {
			logs.Log.Errorf(err.Error())
			return
		}
		err = SaveProfile(p)
		if err != nil {
			logs.Log.Errorf(err.Error())
			return
		}
	}
}

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

func RefreshProfile() error {
	a := &Profile{}
	config.MapStruct("", a)
	err := config.ReloadFiles()
	if err != nil {
		return err
	}
	config.MapStruct("", a)
	return nil
}

func GetProfile() (*Profile, error) {
	p := &Profile{}
	err := config.MapStruct("", p)
	if err != nil {
		return p, err
	}
	return p, nil
}

func GetAliases() ([]string, error) {
	var alias []string
	err := config.MapStruct("aliases", alias)
	if err != nil {
		return alias, err
	}
	return alias, nil
}

func GetExtensions() ([]string, error) {
	var extension []string
	err := config.MapStruct("extensions", extension)
	if err != nil {
		return extension, err
	}
	return extension, nil
}

func GetMals() ([]string, error) {
	var mal []string
	err := config.MapStruct("mals", mal)
	if err != nil {
		return mal, err
	}
	return mal, nil

}

func GetSetting() (*Settings, error) {
	s := &Settings{}
	err := config.MapStruct("settings", s)
	if err != nil {
		return s, err
	}
	return s, nil
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
