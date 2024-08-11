package assets

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/helper"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

var (
	MaliceDirName = ".config/malice"
	ConfigDirName = "configs"
)

func GetConfigDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ConfigDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

func GetRootAppDir() string {
	user, _ := user.Current()
	dir := filepath.Join(user.HomeDir, MaliceDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Error(err.Error())
		}
	}
	return dir
}

func GetConfigs() ([]string, error) {
	var files []string

	// Traverse all files in the specified directory.
	err := filepath.Walk(GetConfigDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			files = append(files, info.Name())
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func MvConfig(oldPath string) error {
	fileName := filepath.Base(oldPath)
	newPath := filepath.Join(GetConfigDir(), fileName)
	err := helper.CopyFile(oldPath, newPath)
	if err != nil {
		return err
	}
	err = helper.RemoveFile(oldPath)
	if err != nil {
		return err
	}
	return nil
}
