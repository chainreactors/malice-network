package mal

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/tui"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var (
	ManifestFileName = "mal.yaml"
)

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

func InstallFromDir(extLocalPath string, promptToOverwrite bool, con *console.Console, isGz bool) {
	var manifestData []byte
	var err error

	if isGz {
		manifestData, err = utils.ReadFileFromTarGz(extLocalPath, fmt.Sprintf("./%s", ManifestFileName))
	} else {
		manifestData, err = os.ReadFile(filepath.Join(extLocalPath, ManifestFileName))
	}
	if err != nil {
		console.Log.Errorf("Error reading %s: %s", ManifestFileName, err)
		return
	}

	manifest, err := ParseMalManifest(manifestData)
	if err != nil {
		console.Log.Errorf("Error parsing %s: %s", ManifestFileName, err)
		return
	}

	installPath := filepath.Join(assets.GetMalsDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			console.Log.Infof("Mal '%s' already exists", manifest.Name)
			confirmModel := tui.NewConfirm("Overwrite current install?")
			newConfirm := tui.NewModel(confirmModel, nil, false, true)
			err = newConfirm.Run()
			if err != nil {
				console.Log.Errorf("Error running confirm model: %s", err)
				return
			}
			if !confirmModel.Confirmed {
				return
			}
		}
		utils.ForceRemoveAll(installPath)
	}

	console.Log.Infof("Installing Mal '%s' (%s) ... ", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0700)
	if err != nil {
		console.Log.Errorf("\nError creating mal directory: %s\n", err)
		return
	}
	err = os.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		console.Log.Errorf("\nFailed to write %s: %s\n", ManifestFileName, err)
		utils.ForceRemoveAll(installPath)
		return
	}
}
