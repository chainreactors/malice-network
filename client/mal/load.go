package mal

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func MalLoadCmd(ctx *cobra.Command, con *console.Console) {
	dirPath := ctx.Flags().Arg(0)
	_, err := LoadMalManiFest(con, dirPath)
	if err != nil {
		console.Log.Error(err)
	}
}

func LoadMalManiFest(con *console.Console, filename string) (*plugin.MalManiFest, error) {
	content, err := os.ReadFile(filepath.Join(assets.GetMalsDir(), filename, ManifestFileName))
	if err != nil {
		return nil, err
	}
	manifest, err := ParseMalManifest(content)
	if err != nil {
		return nil, err
	}

	err = con.Plugins.LoadPlugin(manifest, con)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}
