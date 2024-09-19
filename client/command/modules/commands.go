package modules

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"strings"
)

func Commands(con *repl.Console) []*cobra.Command {
	listModuleCmd := &cobra.Command{
		Use:   consts.ModuleListModule,
		Short: "List modules",
		Long:  help.GetHelpFor(consts.ModuleListModule),
		Run: func(cmd *cobra.Command, args []string) {
			ListModulesCmd(cmd, con)
			return
		},
	}

	loadModuleCmd := &cobra.Command{
		Use:   consts.ModuleLoadModule + " [module_file]",
		Short: "Load module",
		Long:  help.GetHelpFor(consts.ModuleLoadModule),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			LoadModuleCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(loadModuleCmd, nil,
		carapace.ActionValues().Usage("module name"),
		carapace.ActionFiles().Usage("path to the module file"))

	refreshModuleCmd := &cobra.Command{
		Use:   consts.ModuleRefreshModule,
		Short: "Refresh module",
		Long:  help.GetHelpFor(consts.ModuleRefreshModule),
		Run: func(cmd *cobra.Command, args []string) {
			RefreshModuleCmd(cmd, con)
			return
		},
	}

	clearCmd := &cobra.Command{
		Use:   consts.ModuleClear,
		Short: "Clear modules",
		Long:  help.GetHelpFor(consts.ModuleClear),
		Run: func(cmd *cobra.Command, args []string) {
			ClearCmd(cmd, con)
			return
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
