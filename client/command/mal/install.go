package mal

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/mals/m"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

var RepoUrl = "https://github.com/chainreactors/mal-community"
var MalLatest = "latest"

// ExtensionsInstallCmd - Install an extension
func MalInstallCmd(cmd *cobra.Command, con *core.Console) error {
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
		if _, err := InstallMal(RepoUrl, name, version, os.Stdout, malHttpConfig, con); err != nil {
			return err
		}
	} else {
		if _, err := InstallFromDir(localPath, true, con); err != nil {
			return err
		}
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

func InstallFromDir(extLocalPath string, promptToOverwrite bool, con *core.Console) (bool, error) {
	var manifestData []byte
	var err error

	format, err := fileutils.DetectArchiveFormat(extLocalPath)
	if err != nil {
		return false, err
	}
	switch format {
	case fileutils.ArchiveZip:
		manifestData, err = fileutils.ReadFileFromZip(extLocalPath, m.ManifestFileName)
	default:
		manifestData, err = fileutils.ReadFileFromTarGz(extLocalPath, m.ManifestFileName)
	}
	if err != nil {
		return false, err
	}
	manifest, err := plugin.ParseMalManifest(manifestData)
	if err != nil {
		return false, err
	}

	installPath := filepath.Join(assets.GetMalsDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		oldManifestPath := filepath.Join(installPath, m.ManifestFileName)
		oldHash, oldErr := fileutils.CalculateSHA256Checksum(oldManifestPath)
		newHashStr := fileutils.CalculateSHA256Byte(manifestData)
		if oldErr == nil && oldHash == newHashStr {
			con.Log.Infof("Mal '%s' is latest version.\n", manifest.Name)
			return false, nil
		}
		if promptToOverwrite {
			con.Log.Infof("Mal '%s' already exists\n", manifest.Name)
			confirmModel := tui.NewConfirm("Overwrite current install?")
			err = confirmModel.Run()
			if err != nil {
				return false, err
			}
			if !confirmModel.GetConfirmed() {
				return false, nil
			}
		}
		fileutils.ForceRemoveAll(installPath)
	}

	con.Log.Infof("Installing Mal '%s' (%s) ... \n", manifest.Name, manifest.Version)
	err = os.MkdirAll(installPath, 0700)
	if err != nil {
		return false, err
	}
	switch format {
	case fileutils.ArchiveZip:
		err = fileutils.ExtractZipFromFile(extLocalPath, installPath)
	default:
		err = fileutils.ExtractTarGz(extLocalPath, installPath)
	}
	if err != nil {
		fileutils.ForceRemoveAll(installPath)
		return false, err
	}
	if manifest.Lib {
		resourcePath := filepath.Join(installPath, "resources")
		if info, statErr := os.Stat(resourcePath); statErr == nil {
			if info.IsDir() {
				err = fileutils.MoveDirectory(resourcePath, assets.GetResourceDir())
				if err == nil {
					_ = fileutils.ForceRemoveAll(resourcePath)
				}
			} else {
				err = fileutils.MoveFile(resourcePath, filepath.Join(assets.GetResourceDir(), info.Name()))
			}
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}
