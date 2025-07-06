package assets

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/gookit/config/v2"
	"golang.org/x/exp/slices"
)

var (
	maliceProfile = "malice.yaml"
)

var HookFn = func(event string, c *config.Config) {
	if strings.HasPrefix(event, "set.") {
		rootDir, _ := filepath.Abs(GetRootAppDir())
		open, err := os.OpenFile(filepath.Join(rootDir, maliceProfile), os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			logs.Log.Errorf("cannot open config , %s ", err.Error())
			return
		}
		defer open.Close()
		_, err = config.DumpTo(open, config.Yaml)
		if err != nil {
			logs.Log.Errorf("cannot dump config , %s ", err.Error())
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

func LoadProfile() (*Profile, error) {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	malicePath := filepath.Join(rootDir, maliceProfile)
	profile := &Profile{}
	if !fileutils.Exist(malicePath) {
		confStr := configutil.InitDefaultConfig(profile, 0)
		err := os.WriteFile(malicePath, confStr, 0644)
		if err != nil {
			return profile, err
		}
		logs.Log.Warnf("config file not found, created default config %s", malicePath)
	}
	err := configutil.LoadConfig(malicePath, profile)
	if err != nil {
		return profile, err
	}
	return profile, nil
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
	profile, err := GetProfile()
	var alias []string
	if err != nil {
		return alias, err
	}
	alias = profile.Aliases
	return alias, nil
}

func GetExtensions() ([]string, error) {
	profile, err := GetProfile()
	var extensions []string
	if err != nil {
		return extensions, err
	}
	extensions = profile.Extensions
	return extensions, nil
}

func GetMals() ([]string, error) {
	profile, err := GetProfile()
	var mal []string
	if err != nil {
		return mal, err
	}
	mal = profile.Mals
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

func (profile *Profile) AddMal(manifestName string) bool {
	if !slices.Contains(profile.Mals, manifestName) {
		profile.Mals = append(profile.Mals, manifestName)
		config.Set("mals", profile.Mals)
		return true
	}
	return false
}

func (profile *Profile) RemoveMal(manifestName string) bool {
	index := slices.Index(profile.Mals, manifestName)
	if index != -1 {
		profile.Mals = slices.Delete(profile.Mals, index, index+1)
		config.Set("mals", profile.Mals)
		return true
	}
	return false
}

func (profile *Profile) AddAlias(alias string) bool {
	if !slices.Contains(profile.Aliases, alias) {
		profile.Aliases = append(profile.Aliases, alias)
		config.Set("aliases", profile.Aliases)
		return true
	}
	return false
}

func (profile *Profile) RemoveAlias(alias string) bool {
	index := slices.Index(profile.Aliases, alias)
	if index != -1 {
		profile.Aliases = slices.Delete(profile.Aliases, index, index+1)
		config.Set("aliases", profile.Aliases)
		return true
	}
	return false
}

func (profile *Profile) AddExtension(extension string) bool {
	if !slices.Contains(profile.Extensions, extension) {
		profile.Extensions = append(profile.Extensions, extension)
		config.Set("extensions", profile.Extensions)
		return true
	}
	return false
}

func (profile *Profile) RemoveExtension(extension string) bool {
	index := slices.Index(profile.Extensions, extension)
	if index != -1 {
		profile.Extensions = slices.Delete(profile.Extensions, index, index+1)
		config.Set("extensions", profile.Extensions)
		return true
	}
	return false
}
