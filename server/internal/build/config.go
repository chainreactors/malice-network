package build

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/config"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"os"
	"path/filepath"
	"strings"
)

var (
	generateConfig = "config.yaml"
	release        = "release"
	malefic        = "malefic"
)

func DbToConfig(req *clientpb.Generate) error {
	var profileDB models.Profile
	var profile configs.GeneratorConfig
	var err error
	if req.Name != "" {
		profileDB, err = db.GetProfile(req.Name)
		if err != nil {
			return err
		}
	}
	path := filepath.Join(configs.BuildPath, generateConfig)
	err = config.LoadConfig(path, &profile)
	if err != nil {
		return err
	}
	err = profileDB.UpdateGeneratorConfig(profile, req, path)
	if err != nil {
		return err
	}
	return nil
}

func MoveBuildOutput(target, buildType, name string) error {
	var sourcePath string
	var dstPath string
	switch buildType {
	case consts.PE:
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.PEFile)
		dstPath = filepath.Join(configs.BuildOutputPath, name+consts.PEFile)
	case consts.DLL:
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.DllFile)
		dstPath = filepath.Join(configs.BuildOutputPath, name+consts.DllFile)
	case consts.Shellcode:
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.ShellcodeFile)
		dstPath = filepath.Join(configs.BuildOutputPath, name+consts.ShellcodeFile)
	case consts.ELF:
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic)
		dstPath = filepath.Join(configs.BuildOutputPath, name)
	}
	err := file.CopyFile(sourcePath, dstPath)
	if err != nil {
		return err
	}
	return nil
}

func GetOutPutPath(name string) (string, error) {
	dirPath := configs.BuildOutputPath

	var matchedPath string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), name) {
			matchedPath = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if matchedPath != "" {
		return matchedPath, nil
	}

	return "", errors.New("build output file not found")
}
