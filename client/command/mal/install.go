package mal

import (
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var (
	ManifestFileName = "mal.yaml"
)

// ExtensionsInstallCmd - Install an extension
func MalInstallCmd(cmd *cobra.Command, con *repl.Console) {
	localPath := cmd.Flags().Arg(0)
	_, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		con.Log.Errorf("Mal path '%s' does not exist", localPath)
		return
	}
	InstallFromDir(localPath, true, con)
}

func ParseMalManifest(data []byte) (*plugin.MalManiFest, error) {
	extManifest := &plugin.MalManiFest{}
	err := yaml.Unmarshal(data, &extManifest)
	if err != nil {
		return nil, err
	}
	return extManifest, validManifest(extManifest)
}

func validManifest(manifest *plugin.MalManiFest) error {
	if manifest.Name == "" {
		return errors.New("missing `name` field in mal manifest")
	}
	return nil
}

func InstallFromDir(extLocalPath string, promptToOverwrite bool, con *repl.Console) {
	var manifestData []byte
	var err error

	manifestData, err = utils.ReadFileFromTarGz(extLocalPath, ManifestFileName)
	if err != nil {
		con.Log.Errorf("Error reading %s: %s", ManifestFileName, err)
		return
	}

	manifest, err := ParseMalManifest(manifestData)
	if err != nil {
		con.Log.Errorf("Error parsing %s: %s", ManifestFileName, err)
		return
	}

	installPath := filepath.Join(assets.GetMalsDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			con.Log.Infof("Mal '%s' already exists", manifest.Name)
			confirmModel := tui.NewConfirm("Overwrite current install?")
			newConfirm := tui.NewModel(confirmModel, nil, false, true)
			err = newConfirm.Run()
			if err != nil {
				con.Log.Errorf("Error running confirm model: %s", err)
				return
			}
			if !confirmModel.Confirmed {
				return
			}
		}
		utils.ForceRemoveAll(installPath)
	}

	con.Log.Infof("Installing Mal '%s' (%s) ... ", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0700)
	if err != nil {
		con.Log.Errorf("\nError creating mal directory: %s\n", err)
		return
	}
	err = utils.ExtractTarGz(extLocalPath, installPath)
	if err != nil {
		con.Log.Errorf("\nFailed to extract tar.gz to %s: %s\n", installPath, err)
		utils.ForceRemoveAll(installPath)
		return
	}
}
