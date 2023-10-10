package assets

import (
	"github.com/chainreactors/logs"
	"os"
	"os/user"
	"path/filepath"
)

var (
	MaliceDirName = ".malice"
)

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
