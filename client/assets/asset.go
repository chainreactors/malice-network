package assets

import (
	_ "embed"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed _inputrc
var inputrc string

var (
	MaliceDirName   = ".config/malice"
	ConfigDirName   = "configs"
	ResourceDirName = "resource"
	TempDirName     = "temp"
	LogDirName      = "log"
)

func GetConfigDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ConfigDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Errorf(err.Error())
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

func GetResourceDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ResourceDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Errorf(err.Error())
		}
	}
	return dir
}

func GetTempDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, TempDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Errorf(err.Error())
		}
	}
	return dir
}

func GenerateTempFile(filename string) (*os.File, error) {
	return os.Create(filepath.Join(GetTempDir(), filename))
}

func GetLogDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, LogDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Errorf(err.Error())
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
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".auth") {
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
	err := file.CopyFile(oldPath, newPath)
	if err != nil {
		return err
	}
	err = file.RemoveFile(oldPath)
	if err != nil {
		return err
	}
	return nil
}

func SetInputrc() {
	home, _ := os.UserHomeDir()
	inputrcPath := filepath.Join(home, "_inputrc")
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(inputrcPath); os.IsNotExist(err) {
			err = os.WriteFile(inputrcPath, []byte(inputrc), 0644)
			if err != nil {
				logs.Log.Errorf(err.Error())
			}
		}
	}
}
