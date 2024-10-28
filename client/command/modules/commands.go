package modules

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strings"
)

func Commands(con *repl.Console) []*cobra.Command {
	listModuleCmd := &cobra.Command{
		Use:   consts.ModuleListModule,
		Short: "List modules",
		// Long:  help.FormatLongHelp(consts.ModuleListModule),
		RunE: func(cmd *cobra.Command, args []string) error {

			return ListModulesCmd(cmd, con)
		},
	}

	loadModuleCmd := &cobra.Command{
		Use:   consts.ModuleLoadModule + " [module_file]",
		Short: "Load module",
		// Long:  help.FormatLongHelp(consts.ModuleLoadModule),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return LoadModuleCmd(cmd, con)
		},
		Example: `load module from malefic-modules
before loading, you can list the current modules: 
~~~
execute_addon、clear ...
~~~
then you can load module
~~~
load_module <module_file.dll>
~~~
you can see more modules loaded by list_module
~~~
execute_addon、clear 、ps、powerpic...
~~~
`}

	common.BindFlag(loadModuleCmd, func(f *pflag.FlagSet) {
		f.StringP("bundle", "b", "", "bundle name")
	})
	common.BindArgCompletions(loadModuleCmd, nil,
		carapace.ActionFiles().Usage("path to the module file"))

	refreshModuleCmd := &cobra.Command{
		Use:   consts.ModuleRefreshModule,
		Short: "Refresh module",
		// Long:  help.FormatLongHelp(consts.ModuleRefreshModule),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RefreshModuleCmd(cmd, con)
		},
	}

	clearCmd := &cobra.Command{
		Use:   consts.ModuleClear,
		Short: "Clear modules",
		// Long:  help.FormatLongHelp(consts.ModuleClear),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ClearCmd(cmd, con)
		},
	}

	return []*cobra.Command{
		listModuleCmd,
		loadModuleCmd,
		refreshModuleCmd,
		clearCmd,
	}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleListModule,
		ListModules,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			resp := ctx.Spite.GetModules()
			var modules []string
			for module := range resp.GetModules() {
				modules = append(modules, fmt.Sprintf("%s", module))
			}
			return strings.Join(modules, ","), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			modules := content.Spite.GetModules()
			if len(modules.Modules) == 0 {
				return "No modules found.", nil
			}

			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				{Title: "Name", Width: 15},
				{Title: "Help", Width: 30},
			}, true)
			for _, module := range modules.GetModules() {
				row = table.Row{
					module,
					"",
				}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.RegisterImplantFunc(
		consts.ModuleLoadModule,
		LoadModule,
		"",
		nil,
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleRefreshModule,
		refreshModule,
		"",
		nil,
		common.ParseStatus,
		nil)

	//clear
	con.RegisterImplantFunc(
		consts.ModuleClear,
		clearAll,
		"",
		nil,
		common.ParseStatus,
		nil)
}
