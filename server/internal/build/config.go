package build

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
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
	modules        = "modules"
	prelude        = "malefic-prelude"
	pulse          = "malefic-pulse"
)

// GenerateProfile - Generate profile
// first recover profile from database
// then use Generate overwrite profile
func GenerateProfile(req *clientpb.Generate) ([]byte, error) {
	var err error
	profile, err := db.GetProfile(req.ProfileName)
	if err != nil {
		return nil, err
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

func MoveBuildOutput(target, buildType string) (string, string, error) {
	var sourcePath string
	name := encoders.UUID()
	switch {
	case strings.Contains(target, "windows"):
		if buildType == consts.CommandBuildModules {
			sourcePath = filepath.Join(configs.TargetPath, target, release, modules+consts.DllFile)
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

func GetFilePath(name, target, buildType string, isSrdi bool) string {
	if isSrdi {
		return name + consts.ShellcodeFile
	}
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
