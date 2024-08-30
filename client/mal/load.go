package mal

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func MalLoadCmd(ctx *cobra.Command, con *console.Console) {
	dirPath := ctx.Flags().Arg(0)
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
