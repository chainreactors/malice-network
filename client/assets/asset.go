package assets

import (
	_ "embed"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
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
	return ensureDir(filepath.Join(rootDir, ConfigDirName))
}

func GetRootAppDir() string {
	if filepath.IsAbs(MaliceDirName) {
		return ensureDir(MaliceDirName)
	}

	var homeDir string
	currentUser, err := user.Current()
	if err == nil && currentUser != nil && currentUser.HomeDir != "" {
		homeDir = currentUser.HomeDir
	} else {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			logs.Log.Error(err.Error())
			return MaliceDirName
		}
	}
	dir := filepath.Join(homeDir, MaliceDirName)
	return ensureDir(dir)
}

func GetResourceDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	return ensureDir(filepath.Join(rootDir, ResourcesDirName))
}

func GetTempDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	return ensureDir(filepath.Join(rootDir, TempDirName))
}

func GenerateTempFile(sessionId, filename string) (*os.File, error) {
	safeSessionID, err := fileutils.SanitizeBasename(sessionId)
	if err != nil {
		return nil, err
	}
	sessionDir, err := fileutils.SafeJoin(GetTempDir(), safeSessionID)
	if err != nil {
		return nil, err
	}
	if !fileutils.Exist(sessionDir) {
		if err := os.MkdirAll(sessionDir, assetsDirPerm); err != nil {
			logs.Log.Errorf("failed to create session directory: %s", err.Error())
		}
	}
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	ext := filepath.Ext(filename)
	timestampMillis := time.Now().UnixNano() / int64(time.Millisecond)
	seconds := timestampMillis / 1000
	nanoseconds := (timestampMillis % 1000) * int64(time.Millisecond)
	t := time.Unix(seconds, nanoseconds)
	fullPath, err := fileutils.SafeJoin(sessionDir, fmt.Sprintf("%s_%s%s", baseName, t.Format("2006-01-02_15-04-05"), ext))
	if err != nil {
		return nil, err
	}
	return os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, assetsFilePerm)
}

func GetLogDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	return ensureDir(filepath.Join(rootDir, LogDirName))
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
		timestamp := time.Now().Format("20060102_150405")
		backupPath := fmt.Sprintf("%s.%s.backup", newPath, timestamp)
		err := fileutils.CopyFile(newPath, backupPath)
		if err != nil {
			logs.Log.Warnf("failed to backup config file %s: %s", newPath, err.Error())
		} else {
			logs.Log.Warnf("config file %s already exists, backed up to %s", newPath, backupPath)
		}
	}

	err = fileutils.CopyFile(oldPath, newPath)
	if err != nil {
		logs.Log.Warnf("failed to copy config file %s: %s", newPath, err.Error())
		return err
	}
	return nil
}
