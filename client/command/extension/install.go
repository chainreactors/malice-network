package extension

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

// ExtensionsInstallCmd - Install an extension
func ExtensionsInstallCmd(ctx *grumble.Context, con *console.Console) {
	extLocalPath := ctx.Args.String("path")
	fi, err := os.Stat(extLocalPath)
	if os.IsNotExist(err) {
		console.Log.Errorf("Extension path '%s' does not exist", extLocalPath)
		return
	}
	if !fi.IsDir() {
		InstallFromFilePath(extLocalPath, false, con)
	} else {
		installFromDir(extLocalPath, con)
	}
}

// Install an extension from a directory
func installFromDir(extLocalPath string, con *console.Console) {
	manifestData, err := ioutil.ReadFile(filepath.Join(extLocalPath, ManifestFileName))
	if err != nil {
		console.Log.Errorf("Error reading %s: %s", ManifestFileName, err)
		return
	}
	manifest, err := ParseExtensionManifest(manifestData)
	if err != nil {
		console.Log.Errorf("Error parsing %s: %s", ManifestFileName, err)
		return
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		console.Log.Infof("Extension '%s' already exists", manifest.CommandName)
		confirm := false
		prompt := &survey.Confirm{Message: "Overwrite current install?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
		forceRemoveAll(installPath)
	}

	console.Log.Infof("Installing extension '%s' (%s) ... ", manifest.CommandName, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		console.Log.Errorf("\nError creating extension directory: %s\n", err)
		return
	}
	err = ioutil.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		console.Log.Errorf("\nFailed to write %s: %s\n", ManifestFileName, err)
		forceRemoveAll(installPath)
		return
	}

	for _, manifestFile := range manifest.Files {
		if manifestFile.Path != "" {
			src := filepath.Join(extLocalPath, utils.ResolvePath(manifestFile.Path))
			dst := filepath.Join(installPath, utils.ResolvePath(manifestFile.Path))
			err := utils.CopyFile(src, dst)
			if err != nil {
				console.Log.Errorf("\nError copying file '%s' -> '%s': %s\n", src, dst, err)
				forceRemoveAll(installPath)
				return
			}
		}
	}

}

// InstallFromFilePath - Install an extension from a .tar.gz file
func InstallFromFilePath(extLocalPath string, autoOverwrite bool, con *console.Console) *string {
	manifestData, err := utils.ReadFileFromTarGz(extLocalPath, fmt.Sprintf("./%s", ManifestFileName))
	if err != nil {
		console.Log.Errorf("Failed to read %s from '%s': %s\n", ManifestFileName, extLocalPath, err)
		return nil
	}
	manifest, err := ParseExtensionManifest(manifestData)
	if err != nil {
		console.Log.Errorf("Failed to parse %s: %s\n", ManifestFileName, err)
		return nil
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if !autoOverwrite {
			console.Log.Infof("Extension '%s' already exists\n", manifest.CommandName)
			confirm := false
			prompt := &survey.Confirm{Message: "Overwrite current install?"}
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return nil
			}
		}
		forceRemoveAll(installPath)
	}

	console.Log.Infof("Installing extension '%s' (%s) ... ", manifest.CommandName, manifest.Version)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		console.Log.Errorf("\nFailed to create extension directory: %s\n", err)
		return nil
	}
	err = ioutil.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		console.Log.Errorf("\nFailed to write %s: %s\n", ManifestFileName, err)
		forceRemoveAll(installPath)
		return nil
	}
	for _, manifestFile := range manifest.Files {
		if manifestFile.Path != "" {
			err = installArtifact(extLocalPath, installPath, manifestFile.Path, con)
			if err != nil {
				console.Log.Errorf("\nFailed to install file: %s\n", err)
				forceRemoveAll(installPath)
				return nil
			}
		}
	}
	console.Log.Infof("done!\n")
	return &installPath
}

func installArtifact(extGzFilePath string, installPath string, artifactPath string, con *console.Console) error {
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
		err := os.MkdirAll(artifactDir, 0o700)
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(localArtifactPath, data, 0o600)
	if err != nil {
		return err
	}
	return nil
}
