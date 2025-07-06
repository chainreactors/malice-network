package build

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

var (
	generateConfig = "config.yaml"
	release        = "release"
	releaseLto     = "release-lto"
	malefic        = "malefic"
	modules        = "malefic_modules"
	modules3rd     = "malefic_3rd"
	prelude        = "malefic-prelude"
	pulse          = "malefic-pulse"
)

// GenerateProfile - Generate profile
// first recover profile from database
// then use Generate overwrite profile
func GenerateProfile(req *clientpb.BuildConfig) ([]byte, error) {
	var profile *types.ProfileConfig
	var err error
	if req.Type == consts.CommandBuildModules && req.ProfileName == "" {
		profile, err = types.LoadProfile(types.DefaultProfile)
		if err != nil {
			return nil, err
		}
		var profileParams types.ProfileParams
		err = json.Unmarshal(req.ParamsBytes, &profileParams)
		if err != nil {
			return nil, err
		}
		if profileParams.Modules != "" {
			profile.Implant.Modules = strings.Split(profileParams.Modules, ",")
		}
	} else {
		profile, err = db.GetProfile(req.ProfileName)
		if err != nil {
			return nil, err
		}
	}
	path := filepath.Join(configs.SourceCodePath, generateConfig)
	err = db.UpdateGeneratorConfig(req, path, profile)
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(profile)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func WirteProfile(config *types.ProfileConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	path := filepath.Join(configs.SourceCodePath, generateConfig)

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func MoveBuildOutput(target, buildType string, enable3RD bool) (string, string, error) {
	var sourcePath string
	name := encoders.UUID()
	switch {
	case strings.Contains(target, "windows"):
		if buildType == consts.CommandBuildModules {
			if enable3RD {
				sourcePath = filepath.Join(configs.TargetPath, target, release, modules3rd+consts.DllFile)
			} else {
				sourcePath = filepath.Join(configs.TargetPath, target, release, modules+consts.DllFile)
			}
		} else if buildType == consts.CommandBuildPrelude {
			sourcePath = filepath.Join(configs.TargetPath, target, release, prelude+consts.PEFile)
		} else if buildType == consts.CommandBuildPulse {
			sourcePath = filepath.Join(configs.TargetPath, target, release, pulse+consts.PEFile)
		} else {
			sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.PEFile)
		}
	case strings.Contains(target, "darwin"):
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic)
	case strings.Contains(target, "linux"):
		if buildType == consts.CommandBuildPrelude {
			sourcePath = filepath.Join(configs.TargetPath, target, release, prelude)
		} else {
			sourcePath = filepath.Join(configs.TargetPath, target, release, malefic)
		}
	}
	dstPath := filepath.Join(configs.BuildOutputPath, name)
	err := fileutils.MoveFile(sourcePath, dstPath)
	if err != nil {
		return "", "", err
	}
	return sourcePath, dstPath, nil
}

func GetFilePath(name, target, buildType, format string) string {
	switch {
	case strings.Contains(target, "windows"):
		if buildType == consts.CommandBuildModules {
			name = name + consts.DllFile
		} else if buildType == consts.CommandBuildPrelude {
			name = name + consts.PEFile
		} else if buildType == consts.CommandBuildPulse {
			name = name + consts.PEFile
		} else {
			name = name + consts.PEFile
		}
	}
	return name
}
