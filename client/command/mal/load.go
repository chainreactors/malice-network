package mal

import (
	"path/filepath"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/mals/m"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func MalLoadCmd(ctx *cobra.Command, con *repl.Console) error {
	dirPath := ctx.Flags().Arg(0)
	malManager := plugin.GetGlobalMalManager()

	manifestPath := filepath.Join(assets.GetMalsDir(), dirPath, m.ManifestFileName)
	manifest, err := plugin.LoadMalManiFest(manifestPath)
	if err != nil {
		return err
	}

	var plug plugin.Plugin

	// 检查是否已加载
	if _, exists := malManager.GetExternalPlugin(manifest.Name); exists {
		con.Log.Warnf("mal %s already loaded, reloading\n", manifest.Name)
		err := malManager.ReloadExternalMal(manifest.Name)
		if err != nil {
			return err
		}
		// 重新获取插件
		plug, _ = malManager.GetExternalPlugin(manifest.Name)
	} else {
		// 首次加载
		plug, err = malManager.LoadExternalMal(manifest)
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
	err = assets.SaveProfile(profile)
	if err != nil {
		return err
	}

	con.Log.Importantf("load mal: %s successfully\n", manifest.Name)
	return nil
}

func LoadMal(con *repl.Console, rootCmd *cobra.Command, filename string) error {
	manifest, err := plugin.LoadMalManiFest(filename)
	if err != nil {
		return err
	}
	return LoadMalWithManifest(con, rootCmd, manifest)
}

func LoadMalWithManifest(con *repl.Console, rootCmd *cobra.Command, manifest *plugin.MalManiFest) error {
	malManager := plugin.GetGlobalMalManager()
	plug, err := malManager.LoadExternalMal(manifest)
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

	err = assets.SaveProfile(profile)
	if err != nil {
		return err
	}
	con.Log.Importantf("load mal: %s successfully\n", manifest.Name)
	return nil
}

func ListMalManifest(con *repl.Console) {
	malManager := plugin.GetGlobalMalManager()

	// 获取所有外部插件
	externalPlugins := malManager.GetAllExternalPlugins()
	embeddedPlugins := malManager.GetAllEmbeddedPlugins()

	if len(externalPlugins) == 0 && len(embeddedPlugins) == 0 {
		con.Log.Infof("No mal loaded")
		return
	}

	rows := []table.Row{}
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 25),
		table.NewColumn("Type", "Type", 8),
		table.NewColumn("Version", "Version", 7),
		table.NewColumn("Author", "Author", 20),
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
	err := tableModel.Run()
	if err != nil {
		con.Log.Errorf("Error running table: %s", err)
		return
	}
}
