package build

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
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

func GenerateProfile(req *clientpb.Generate) (string, error) {
	var err error
	profile, err := db.GetProfile(req.ProfileName)
	if err != nil {
		return "", err
	}
	path := filepath.Join(configs.SourceCodePath, generateConfig)

	return db.UpdateGeneratorConfig(req, path, profile)
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
			sourcePath = filepath.Join(configs.TargetPath, target, releaseLto, pulse+consts.PEFile)
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

func GetFilePath(name, target string) string {
	switch {
	case strings.Contains(target, "windows"):
		if target == consts.CommandBuildModules {
			name = name + consts.DllFile
		} else if target == consts.CommandBuildPrelude {
			name = name + consts.PEFile
		} else if target == consts.CommandBuildPulse {
			name = name + consts.PEFile
		} else {
			name = name + consts.PEFile
		}
	}
	return name
}
