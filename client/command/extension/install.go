package extension

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/tui"
	"os"
	"path/filepath"
	"strings"
)

// ExtensionsInstallCmd - Install an extension
func ExtensionsInstallCmd(ctx *grumble.Context, con *console.Console) {
	extLocalPath := ctx.Args.String("path")
	_, err := os.Stat(extLocalPath)
	if os.IsNotExist(err) {
		console.Log.Errorf("Extension path '%s' does not exist", extLocalPath)
		return
	}
	InstallFromDir(extLocalPath, true, con, strings.HasSuffix(extLocalPath, ".tar.gz"))
}

// Install an extension from a directory
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

	manifest, err := ParseExtensionManifest(manifestData)
	if err != nil {
		console.Log.Errorf("Error parsing %s: %s", ManifestFileName, err)
		return
	}

	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			console.Log.Infof("Extension '%s' already exists", manifest.Name)
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

	console.Log.Infof("Installing extension '%s' (%s) ... ", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0700)
	if err != nil {
		console.Log.Errorf("\nError creating extension directory: %s\n", err)
		return
	}
	err = os.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		console.Log.Errorf("\nFailed to write %s: %s\n", ManifestFileName, err)
		utils.ForceRemoveAll(installPath)
		return
	}
	for _, manifestCmd := range manifest.ExtCommand {
		newInstallPath := filepath.Join(installPath)
		for _, manifestFile := range manifestCmd.Files {
			if manifestFile.Path != "" {
				if isGz {
					err = utils.InstallArtifact(extLocalPath, newInstallPath, manifestFile.Path)
				} else {
					src := filepath.Join(extLocalPath, utils.ResolvePath(manifestFile.Path))
					dst := filepath.Join(newInstallPath, utils.ResolvePath(manifestFile.Path))
					err = os.MkdirAll(filepath.Dir(dst), 0700) //required for extensions with multiple dirs between the .o file and the manifest
					if err != nil {
						console.Log.Errorf("\nError creating extension directory: %s\n", err)
						utils.ForceRemoveAll(newInstallPath)
						return
					}
					err = utils.CopyFile(src, dst)
					if err != nil {
						err = fmt.Errorf("error copying file '%s' -> '%s': %s", src, dst, err)
					}
				}
				if err != nil {
					console.Log.Errorf("Error installing command: %s\n", err)
					utils.ForceRemoveAll(newInstallPath)
					return
				}
			}
		}
	}
}
