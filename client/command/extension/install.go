package extension

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// ExtensionsInstallCmd - Install an extension
func ExtensionsInstallCmd(cmd *cobra.Command, con *console.Console) {
	extLocalPath := cmd.Flags().Arg(0)
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
					err = installArtifact(extLocalPath, newInstallPath, manifestFile.Path)
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

// InstallFromFilePath - Install an extension from a .tar.gz file
//func InstallFromFilePath(extLocalPath string, promptToOverwrite bool, isGz bool) *string {
//	var manifestData []byte
//	var err error
//
//	if isGz {
//		manifestData, err = utils.ReadFileFromTarGz(extLocalPath, fmt.Sprintf("./%s", ManifestFileName))
//	} else {
//		manifestData, err = os.ReadFile(filepath.Join(extLocalPath, ManifestFileName))
//	}
//	if err != nil {
//		console.Log.Errorf("Error reading %s: %s", ManifestFileName, err)
//		return nil
//	}
//
//	manifestF, err := ParseExtensionManifest(manifestData)
//	if err != nil {
//		console.Log.Errorf("Failed to parse %s: %s\n", ManifestFileName, err)
//		return nil
//	}
//	minstallPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifestF.Name))
//	if _, err := os.Stat(minstallPath); !os.IsNotExist(err) {
//		if promptToOverwrite {
//			console.Log.Infof("Extension '%s' already exists\n", manifestF.Name)
//			confirmModel := tui.NewConfirm("Overwrite current install?")
//			err := tui.Run(confirmModel)
//			if err != nil {
//				console.Log.Errorf("Error running confirm model: %s", err)
//				return nil
//			}
//			if !confirmModel.Confirmed {
//				return nil
//			}
//		}
//		forceRemoveAll(minstallPath)
//	}
//
//	console.Log.Infof("Installing extension '%s' (%s) ... ", manifestF.Name, manifestF.Version)
//	err = os.MkdirAll(minstallPath, 0o700)
//	if err != nil {
//		console.Log.Errorf("\nFailed to create extension directory: %s\n", err)
//		return nil
//	}
//	err = os.WriteFile(filepath.Join(minstallPath, ManifestFileName), manifestData, 0o600)
//	if err != nil {
//		console.Log.Errorf("\nFailed to write %s: %s\n", ManifestFileName, err)
//		forceRemoveAll(minstallPath)
//		return nil
//	}
//	for _, manifest := range manifestF.ExtCommand {
//		installPath := filepath.Join(minstallPath)
//		for _, manifestFile := range manifest.Files {
//			if isGz {
//				err = installArtifact(extLocalPath, installPath, manifestFile.Path)
//			} else {
//
//			}
//
//		}
//		if manifest.Path != "" {
//			err = installArtifact(extLocalPath, minstallPath, manifest.Path)
//			if err != nil {
//				console.Log.Errorf("\nFailed to install file: %s\n", err)
//				forceRemoveAll(minstallPath)
//				return nil
//			}
//		}
//	}
//	console.Log.Infof("done!\n")
//	return &minstallPath
//}

func installArtifact(extGzFilePath string, installPath string, artifactPath string) error {
	data, err := utils.ReadFileFromTarGz(extGzFilePath, "."+artifactPath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("archive path '%s' is empty", "."+artifactPath)
	}
	localArtifactPath := filepath.Join(installPath, utils.ResolvePath(artifactPath))
	artifactDir := filepath.Dir(localArtifactPath)
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		err := os.MkdirAll(artifactDir, 0700)
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(localArtifactPath, data, 0600)
	if err != nil {
		return err
	}
	return nil
}
