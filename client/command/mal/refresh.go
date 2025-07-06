package mal

import (
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func RefreshMalCmd(cmd *cobra.Command, con *repl.Console) error {
	malManager := plugin.GetGlobalMalManager()
	implantCmd := con.ImplantMenu()

	// 移除所有mal组的命令
	for _, c := range implantCmd.Commands() {
		if c.GroupID == consts.MalGroup {
			implantCmd.RemoveCommand(c)
		}
	}

	// 获取所有外部插件名称
	externalPlugins := malManager.GetAllExternalPlugins()
	var pluginNames []string
	for name := range externalPlugins {
		pluginNames = append(pluginNames, name)
	}

	// 移除所有外部插件
	for _, name := range pluginNames {
		err := malManager.RemoveExternalMal(name)
		if err != nil {
			con.Log.Warnf("Failed to remove plugin %s: %s\n", name, err)
		}
	}

	// 重新加载所有外部mal插件
	for _, manifest := range malManager.GetPluginManifests() {
		err := LoadMalWithManifest(con, implantCmd, manifest)
		if err != nil {
			con.Log.Errorf("Failed to load mal %s: %s\n", manifest.Name, err)
			continue
		}
	}

	con.Log.Importantf("Successfully refreshed all mal plugins\n")
	return nil
}
