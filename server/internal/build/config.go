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

func GenerateProfile(req *clientpb.Generate) error {
	var err error
	profile, err := db.GetProfile(req.ProfileName)
	if err != nil {
		return err
	}
	path := filepath.Join(configs.SourceCodePath, generateConfig)

	return db.UpdateGeneratorConfig(req, path, profile)
}

func MoveBuildOutput(target, buildType string) (string, string, error) {
	var sourcePath string
	var dstPath string
	name := encoders.UUID()
	switch {
	case strings.Contains(target, "windows"):
		if buildType == consts.CommandBuildModules {
			sourcePath = filepath.Join(configs.TargetPath, target, release, modules+consts.DllFile)
			dstPath = filepath.Join(configs.BuildOutputPath, name+consts.DllFile)
		} else if buildType == consts.CommandBuildPrelude {
			sourcePath = filepath.Join(configs.TargetPath, target, release, prelude+consts.PEFile)
			dstPath = filepath.Join(configs.BuildOutputPath, name+consts.PEFile)
		} else if buildType == consts.CommandBuildPulse {
			sourcePath = filepath.Join(configs.TargetPath, target, releaseLto, pulse+consts.PEFile)
			dstPath = filepath.Join(configs.BuildOutputPath, name+consts.PEFile)
		} else {
			sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.PEFile)
			dstPath = filepath.Join(configs.BuildOutputPath, name+consts.PEFile)
		}
	case strings.Contains(target, "darwin"):
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic)
		dstPath = filepath.Join(configs.BuildOutputPath, name)
	case strings.Contains(target, "linux"):
		if buildType == consts.CommandBuildPrelude {
			sourcePath = filepath.Join(configs.TargetPath, target, release, prelude)
			dstPath = filepath.Join(configs.BuildOutputPath, name)
		} else {
			sourcePath = filepath.Join(configs.TargetPath, target, release, malefic)
			dstPath = filepath.Join(configs.BuildOutputPath, name)
		}
	}
	err := fileutils.MoveFile(sourcePath, dstPath)
	if err != nil {
		return "", "", err
	}
	return sourcePath, dstPath, nil
}
