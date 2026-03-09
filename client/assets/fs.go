package assets

import (
	"os"

	"github.com/chainreactors/logs"
)

const (
	assetsDirPerm  = 0o700
	assetsFilePerm = 0o600
)

func ensureDir(path string) string {
	if err := os.MkdirAll(path, assetsDirPerm); err != nil {
		logs.Log.Errorf("%v", err)
	}
	return path
}

