package alias

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

// AliasesInstallCmd - Install an alias
func AliasesInstallCmd(cmd *cobra.Command, con *repl.Console) {
	aliasLocalPath := cmd.Flags().Arg(0)
	fi, err := os.Stat(aliasLocalPath)
	if os.IsNotExist(err) {
		con.Log.Errorf("alias path '%s' does not exist\n", aliasLocalPath)
		return
	}
	if !fi.IsDir() {
		InstallFromFile(aliasLocalPath, "", false, con)
	} else {
		installFromDir(aliasLocalPath, con)
	}
}

// Install an extension from a directory
func installFromDir(aliasLocalPath string, con *repl.Console) {
	manifestData, err := os.ReadFile(filepath.Join(aliasLocalPath, ManifestFileName))
	if err != nil {
		con.Log.Errorf("Error reading %s: %s\n", ManifestFileName, err)
		return
	}
	manifest, err := ParseAliasManifest(manifestData)
	if err != nil {
		con.Log.Errorf("Error parsing %s: %s\n", ManifestFileName, err)
		return
	}
	installPath := filepath.Join(assets.GetAliasesDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		con.Log.Infof("Alias '%s' already exists\n", manifest.CommandName)
		//confirm := false
		// todo rewrite to tea
		//prompt := &survey.Confirm{Message: "Overwrite current install?"}
		//survey.AskOne(prompt, &confirm)
		//if !confirm {
		//	return
		//}
		fileutils.ForceRemoveAll(installPath)
	}

	con.Log.Infof("Installing alias '%s' (%s) ... \n", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0700)
	if err != nil {
		con.Log.Errorf("Error creating alias directory: %s\n", err)
		return
	}
	err = os.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.Log.Errorf("Failed to write %s: %s\n", ManifestFileName, err)
		fileutils.ForceRemoveAll(installPath)
		return
	}

	for _, cmdFile := range manifest.Files {
		if cmdFile.Path != "" {
			src := filepath.Join(aliasLocalPath, fileutils.ResolvePath(cmdFile.Path))
			dst := filepath.Join(installPath, fileutils.ResolvePath(cmdFile.Path))
			err := fileutils.CopyFile(src, dst)
			if err != nil {
				con.Log.Errorf("Error copying file '%s' -> '%s': %s\n", src, dst, err)
				fileutils.ForceRemoveAll(installPath)
				return
			}
		}
	}

	con.Log.Infof("done!\n")
}

// Install an extension from a .tar.gz file
func InstallFromFile(aliasGzFilePath string, aliasName string, promptToOverwrite bool, con *repl.Console) *string {
	manifestData, err := fileutils.ReadFileFromTarGz(aliasGzFilePath, fmt.Sprintf("./%s", ManifestFileName))
	if err != nil {
		con.Log.Errorf("Failed to read %s from '%s': %s\n", ManifestFileName, aliasGzFilePath, err)
		return nil
	}
	manifest, err := ParseAliasManifest(manifestData)
	if err != nil {
		errorMsg := ""
		if aliasName != "" {
			errorMsg = fmt.Sprintf("Error processing manifest for alias %s - failed to parse %s: %s\n", aliasName, ManifestFileName, err)
		} else {
			errorMsg = fmt.Sprintf("Failed to parse %s: %s\n", ManifestFileName, err)
		}
		con.Log.Errorf(errorMsg + "\n")
		return nil
	}
	installPath := filepath.Join(assets.GetAliasesDir(), filepath.Base(manifest.CommandName))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			con.Log.Infof("Alias '%s' already exists\n", manifest.CommandName)
			confirmModel := tui.NewConfirm("Overwrite current install?")
			confirmModel.SetHandle(func() {
				fileutils.ForceRemoveAll(installPath)
			})
			err := confirmModel.Run()
			if err != nil {
				con.Log.Errorf("Failed to run confirm model: %s\n", err)
				return nil
			}
			if !confirmModel.GetConfirmed() {
				return nil
			}
		}
	}

	con.Log.Infof("Installing alias '%s' (%s) ... \n", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0700)
	if err != nil {
		con.Log.Errorf("Failed to create alias directory: %s\n", err)
		return nil
	}
	err = os.WriteFile(filepath.Join(installPath, ManifestFileName), manifestData, 0o600)
	if err != nil {
		con.Log.Errorf("Failed to write %s: %s\n", ManifestFileName, err)
		fileutils.ForceRemoveAll(installPath)
		return nil
	}
	for _, aliasFile := range manifest.Files {
		if aliasFile.Path != "" {
			err := InstallAlias(aliasGzFilePath, installPath, aliasFile.Path)
			if err != nil {
				con.Log.Errorf("Failed to install file: %s\n", err)
				fileutils.ForceRemoveAll(installPath)
				return nil
			}
		}
	}
	con.Log.Console("done!\n")
	return &installPath
}

func InstallAlias(aliasGzFilePath string, installPath, artifactPath string) error {
	data, err := fileutils.ReadFileFromTarGz(aliasGzFilePath, fmt.Sprintf("./%s", strings.TrimPrefix(artifactPath, string(os.PathSeparator))))
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("empty file %s", artifactPath)
	}
	localArtifactPath := filepath.Join(installPath, fileutils.ResolvePath(artifactPath))
	artifactDir := filepath.Dir(localArtifactPath)
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		os.MkdirAll(artifactDir, 0700)
	}
	err = os.WriteFile(localArtifactPath, data, 0700)
	if err != nil {
		return err
	}
	return nil
}
