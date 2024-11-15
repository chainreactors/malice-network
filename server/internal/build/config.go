package build

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/config"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"path/filepath"
	"strings"
)

var (
	generateConfig = "config.yaml"
	release        = "release"
	malefic        = "malefic"
	modules        = "modules"
)

func DbToConfig(req *clientpb.Generate) error {
	var profileDB models.Profile
	var profile configs.GeneratorConfig
	var err error
	if req.ProfileName != "" {
		profileDB, err = db.GetProfile(req.ProfileName)
		if err != nil {
			return err
		}
	}
	path := filepath.Join(configs.SourceCodePath, generateConfig)
	err = config.LoadConfig(path, &profile)
	if err != nil {
		return err
	}

	return db.UpdateGeneratorConfig(req, path, profileDB)
}

func MoveBuildOutput(target, buildType string) (string, string, error) {
	var sourcePath string
	var dstPath string
	name := encoders.UUID()
	switch {
	case strings.Contains(target, "windows"):
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.PEFile)
		dstPath = filepath.Join(configs.BuildOutputPath, name)
		if buildType == consts.CommandBuildModules {
			sourcePath = filepath.Join(configs.TargetPath, target, release, modules+consts.DllFile)
			dstPath = filepath.Join(configs.BuildOutputPath, name+consts.DllFile)
		}
	//case consts.DLL:
	//	sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.DllFile)
	//	dstPath = filepath.Join(configs.BuildOutputPath, name+consts.DllFile)
	//case consts.Shellcode:
	//	sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.ShellcodeFile)
	//	dstPath = filepath.Join(configs.BuildOutputPath, name+consts.ShellcodeFile)
	case strings.Contains(target, "linux"):
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic)
		dstPath = filepath.Join(configs.BuildOutputPath, name)
	}
	err := fileutils.CopyFile(sourcePath, dstPath)
	if err != nil {
		return "", "", err
	}
	return sourcePath, dstPath, nil
}
