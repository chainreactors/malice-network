package assets

import (
	_ "embed"
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

//go:embed audit.html
var AuditHtml []byte

var (
	MaliceDirName    = ".config/malice"
	ConfigDirName    = "configs"
	ResourcesDirName = "resources"
	TempDirName      = "temp"
	LogDirName       = "log"
)

func init() {
	InitLogDir()
}

func GetConfigDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ConfigDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Errorf("%v", err)
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
	dir := filepath.Join(rootDir, ResourcesDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Errorf("%s", err.Error())
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
			logs.Log.Errorf("%s", err.Error())
		}
	}
	return dir
}

func GenerateTempFile(sessionId, filename string) (*os.File, error) {
	sessionDir := filepath.Join(GetTempDir(), sessionId)
	if !fileutils.Exist(sessionDir) {
		if err := os.MkdirAll(sessionDir, os.ModePerm); err != nil {
			logs.Log.Errorf("failed to create session directory: %s", err.Error())
		}
	}
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	ext := filepath.Ext(filename)
	fullPath := filepath.Join(sessionDir, filename)
	timestampMillis := time.Now().UnixNano() / int64(time.Millisecond)
	seconds := timestampMillis / 1000
	nanoseconds := (timestampMillis % 1000) * int64(time.Millisecond)
	t := time.Unix(seconds, nanoseconds)
	fullPath = filepath.Join(sessionDir, fmt.Sprintf("%s_%s%s", baseName, t.Format("2006-01-02_15-04-05"), ext))
	return os.Create(fullPath)
}

func GetLogDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, LogDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Errorf("%s", err.Error())
		}
	}
	return dir
}

// InitLogDir initializes the log directory for core.Session
func InitLogDir() {
	client.LogDir = GetLogDir()
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

func LoadConfig(filename string) (*mtls.ClientConfig, error) {
	if fileutils.Exist(filename) {
		err := MvConfig(filename)
		if err != nil {
			return nil, err
		}
	}
	baseFilename := filepath.Base(filename)
	configPath := filepath.Join(GetConfigDir(), baseFilename)
	if fileutils.Exist(configPath) {
		filename = configPath
	} else {
		return nil, fmt.Errorf("config file %s not found", filename)
	}

	config, err := mtls.ReadConfig(filename)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func MvConfig(oldPath string) error {
	fileName := filepath.Base(oldPath)
	newPath := filepath.Join(GetConfigDir(), fileName)

	// Check if source and destination are the same to avoid unnecessary operations
	oldPathAbs, err := filepath.Abs(oldPath)
	if err != nil {
		return err
	}
	newPathAbs, err := filepath.Abs(newPath)
	if err != nil {
		return err
	}
	if oldPathAbs == newPathAbs {
		// File is already in the correct location, no need to move
		return nil
	}

	// Backup existing file if it exists
	if fileutils.Exist(newPath) {
		backupPath := newPath + ".backup"
		err := fileutils.CopyFile(newPath, backupPath)
		if err != nil {
			logs.Log.Warnf("failed to backup config file %s: %s", newPath, err.Error())
		} else {
			logs.Log.Warnf("config file %s already exists, backed up to %s", newPath, backupPath)
		}
	}

	// Use MoveFile which handles overwriting automatically via os.Create
	return fileutils.MoveFile(oldPath, newPath)
}
