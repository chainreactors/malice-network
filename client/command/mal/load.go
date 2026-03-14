package mal

import (
	"fmt"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"path/filepath"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/mals/m"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ensureMalManager(con *core.Console) (*plugin.MalManager, error) {
	if con == nil {
		return nil, fmt.Errorf("console not initialized")
	}
	if con.MalManager == nil {
		con.MalManager = plugin.GetGlobalMalManager()
	}
	return con.MalManager, nil
}

func MalLoadCmd(ctx *cobra.Command, con *core.Console) error {
	manager, err := ensureMalManager(con)
	if err != nil {
		return err
	}

	dirPath := ctx.Flags().Arg(0)
	manifestPath := filepath.Join(assets.GetMalsDir(), dirPath, m.ManifestFileName)
	manifest, err := plugin.LoadMalManiFest(manifestPath)
	if err != nil {
		return err
	}

	var plug plugin.Plugin

	// 检查是否已加载
	if _, exists := manager.GetExternalPlugin(manifest.Name); exists {
		con.Log.Warnf("mal %s already loaded, reloading\n", manifest.Name)
		err := manager.ReloadExternalMal(manifest.Name)
		if err != nil {
			return err
		}
		// 重新获取插件
		plug, _ = manager.GetExternalPlugin(manifest.Name)
	} else {
		// 首次加载
		plug, err = manager.LoadExternalMal(manifest)
		if err != nil {
			return err
		}
	}

	// 添加事件钩子
	for event, fn := range plug.GetEvents() {
		con.AddEventHook(event, fn)
	}

	// 添加命令到implant菜单
	for _, cmd := range plug.Commands() {
		con.ImplantMenu().AddCommand(cmd.Command)
		logs.Log.Debugf("add command: %s", cmd.Command.Name())
	}

	// 更新配置文件
	profile, err := assets.GetProfile()
	if err != nil {
		return err
	}
	profile.AddMal(manifest.Name)
	con.Log.Importantf("load mal: %s successfully\n", manifest.Name)
	return nil
}

func LoadMal(con *core.Console, rootCmd *cobra.Command, filename string) error {
	manifest, err := plugin.LoadMalManiFest(filename)
	if err != nil {
		return err
	}
	return LoadMalWithManifest(con, rootCmd, manifest)
}

func LoadMalWithManifest(con *core.Console, rootCmd *cobra.Command, manifest *plugin.MalManiFest) error {
	manager, err := ensureMalManager(con)
	if err != nil {
		return err
	}

	plug, err := manager.LoadExternalMal(manifest)
	if err != nil {
		return err
	}

	// 添加事件钩子
	for event, fn := range plug.GetEvents() {
		con.AddEventHook(event, fn)
	}

	// 更新配置文件
	profile, err := assets.GetProfile()
	if err != nil {
		return err
	}
	profile.AddMal(manifest.Name)

	// 注册命令
	for _, cmd := range plug.Commands() {
		rootCmd.AddCommand(cmd.Command)
	}
	con.Log.Importantf("load mal: %s successfully\n", manifest.Name)
	return nil
}

func ListMalManifest(cmd *cobra.Command, con *core.Console) {
	manager, err := ensureMalManager(con)
	if err != nil {
		con.Log.Errorf("%s\n", err)
		return
	}

	// 获取所有外部插件
	externalPlugins := manager.GetAllExternalPlugins()
	embeddedPlugins := manager.GetAllEmbeddedPlugins()

	if len(externalPlugins) == 0 && len(embeddedPlugins) == 0 {
		con.Log.Infof("No mal loaded")
		return
	}

	rows := []table.Row{}
	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("Name", "Name", 1),
		table.NewColumn("Type", "Type", 8),
		table.NewColumn("Version", "Version", 7),
		table.NewFlexColumn("Author", "Author", 1),
		table.NewColumn("Source", "Source", 10),
	}, true)

	// 添加嵌入式插件
	for _, plug := range embeddedPlugins {
		manifest := plug.Manifest()
		row := table.NewRow(
			table.RowData{
				"Name":    manifest.Name,
				"Type":    manifest.Type,
				"Version": manifest.Version,
				"Author":  manifest.Author,
				"Source":  "embedded",
			},
		)
		rows = append(rows, row)
	}

	// 添加外部插件
	for _, plug := range externalPlugins {
		manifest := plug.Manifest()
		row := table.NewRow(
			table.RowData{
				"Name":    manifest.Name,
				"Type":    manifest.Type,
				"Version": manifest.Version,
				"Author":  manifest.Author,
				"Source":  "external",
			},
		)
		rows = append(rows, row)
	}

	tableModel.SetRows(rows)
	tableModel.SetMultiline()
	if common.ShouldUseStaticOutput(cmd) {
		con.Log.Console(tableModel.View())
		return
	}

	err = tableModel.Run()
	if err != nil {
		con.Log.Errorf("Error running table: %s", err)
		return
	}
}
