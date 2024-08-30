package mal

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"os"
	"path/filepath"
)

func MalLoadCmd(ctx *grumble.Context, con *console.Console) {
	dirPath := ctx.Args.String("dir-path")
	content, err := os.ReadFile(filepath.Join(assets.GetMalsDir(), dirPath, ManifestFileName))
	if err != nil {
		console.Log.Errorf(err.Error())
		return
	}
	manifest, err := ParseMalManifest(content)
	if err != nil {
		return
	}

	err = con.Plugins.LoadPlugin(manifest, con)
	if err != nil {
		console.Log.Errorf(err.Error())
		return
	}
}
