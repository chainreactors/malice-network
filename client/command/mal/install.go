package mal

import (
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var (
	ManifestFileName = "mal.yaml"
)

// ExtensionsInstallCmd - Install an extension
func MalInstallCmd(cmd *cobra.Command, con *repl.Console) error {
	localPath := cmd.Flags().Arg(0)
	_, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		return errors.New("file does not exist")
	}
	InstallFromDir(localPath, true, con)
	return nil
}

func InstallFromDir(extLocalPath string, promptToOverwrite bool, con *repl.Console) {
	var manifestData []byte
	var err error

	manifestData, err = file.ReadFileFromTarGz(extLocalPath, ManifestFileName)
	if err != nil {
		con.Log.Errorf("Error reading %s: %s\n", ManifestFileName, err)
		return
	}

	manifest, err := plugin.ParseMalManifest(manifestData)
	if err != nil {
		con.Log.Errorf("Error parsing %s: %s\n", ManifestFileName, err)
		return
	}

	installPath := filepath.Join(assets.GetMalsDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			con.Log.Infof("Mal '%s' already exists\n", manifest.Name)
			confirmModel := tui.NewConfirm("Overwrite current install?")
			newConfirm := tui.NewModel(confirmModel, nil, false, true)
			err = newConfirm.Run()
			if err != nil {
				con.Log.Errorf("Error running confirm model: %s\n", err)
				return
			}
			if !confirmModel.Confirmed {
				return
			}
		}
		file.ForceRemoveAll(installPath)
	}

	con.Log.Infof("Installing Mal '%s' (%s) ... \n", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0700)
	if err != nil {
		con.Log.Errorf("\nError creating mal directory: %s\n", err)
		return
	}
	err = file.ExtractTarGz(extLocalPath, installPath)
	if err != nil {
		con.Log.Errorf("\nFailed to extract tar.gz to %s: %s\n", installPath, err)
		file.ForceRemoveAll(installPath)
		return
	}
}
