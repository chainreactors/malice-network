package mal

import (
	"github.com/chainreactors/malice-network/client/plugin"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/mals/m"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

var repoUrl = "https://github.com/chainreactors/mal-community"

// ExtensionsInstallCmd - Install an extension
func MalInstallCmd(cmd *cobra.Command, con *repl.Console) error {
	localPath := cmd.Flags().Arg(0)
	malHttpConfig := parseMalHTTPConfig(cmd)
	version, _ := cmd.Flags().GetString("version")
	_, err := os.Stat(localPath)
	filename := filepath.Base(localPath)

	// 去除双重扩展名，.tar.gz
	name := strings.TrimSuffix(filename, filepath.Ext(filename)) // 去除 .gz，结果是 common.tar
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if os.IsNotExist(err) {
		malsJson, err := m.ParserMalYaml(m.DefaultMalRepoURL, assets.GetConfigDir(), malHttpConfig)
		if err != nil {
			return err
		}
		if version == "latest" {
			for _, mal := range malsJson.Mals {
				if mal.Name == name {
					version = mal.Version
					break
				}
			}
		}
		InstallMal(repoUrl, name, version, os.Stdout, malHttpConfig, con)
	} else {
		InstallFromDir(localPath, true, con)
	}

	// 使用统一的MalManager加载插件
	err = LoadMalWithManifest(con, con.ImplantMenu(), &plugin.MalManiFest{Name: name})
	if err != nil {
		// 如果直接加载失败，尝试从manifest文件加载
		manifestPath := filepath.Join(assets.GetMalsDir(), name, m.ManifestFileName)
		manifest, manifestErr := plugin.LoadMalManiFest(manifestPath)
		if manifestErr != nil {
			return manifestErr
		}

		err = LoadMalWithManifest(con, con.ImplantMenu(), manifest)
		if err != nil {
			return err
		}
	}

	con.Log.Importantf("Successfully installed and loaded mal: %s\n", name)
	return nil
}

func InstallFromDir(extLocalPath string, promptToOverwrite bool, con *repl.Console) {
	var manifestData []byte
	var err error

	manifestData, err = fileutils.ReadFileFromTarGz(extLocalPath, m.ManifestFileName)
	if err != nil {
		con.Log.Errorf("Error reading %s: %s\n", m.ManifestFileName, err)
		return
	}

	manifest, err := plugin.ParseMalManifest(manifestData)
	if err != nil {
		con.Log.Errorf("Error parsing %s: %s\n", m.ManifestFileName, err)
		return
	}

	installPath := filepath.Join(assets.GetMalsDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		if promptToOverwrite {
			con.Log.Infof("Mal '%s' already exists\n", manifest.Name)
			confirmModel := tui.NewConfirm("Overwrite current install?")
			err = confirmModel.Run()
			if err != nil {
				con.Log.Errorf("Error running confirm model: %s\n", err)
				return
			}
			if !confirmModel.GetConfirmed() {
				return
			}
		}
		fileutils.ForceRemoveAll(installPath)
	}

	con.Log.Infof("Installing Mal '%s' (%s) ... \n", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0700)
	if err != nil {
		con.Log.Errorf("\nError creating mal directory: %s\n", err)
		return
	}
	err = fileutils.ExtractTarGz(extLocalPath, installPath)
	if err != nil {
		con.Log.Errorf("\nFailed to extract tar.gz to %s: %s\n", installPath, err)
		fileutils.ForceRemoveAll(installPath)
		return
	}
	if manifest.Lib {
		err := fileutils.MoveFile(filepath.Join(installPath, "resources"), assets.GetResourceDir())
		if err != nil {
			con.Log.Errorf("\nFailed to move resources to %s: %s\n", assets.GetResourceDir(), err)
			return
		}
	}
}
