package modules

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"
)

func Commands(con *core.Console) []*cobra.Command {
	moduleCmd := &cobra.Command{
		Use:   consts.CommandModule,
		Short: "Module management",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListModulesCmd(cmd, con)
		},
	}

	loadCmd := &cobra.Command{
		Use:   "load [module_file]",
		Short: "Load module",
		RunE: func(cmd *cobra.Command, args []string) error {
			return LoadModuleCmd(cmd, con)
		},
		Example: `load module from malefic-modules
before loading, you can list the current modules:
~~~
module list
~~~
then you can load module
~~~
module load --path <module_file.dll>
~~~
you can see more modules loaded by module list
~~~
execute_addon,clear,ps,powershell...
~~~
`,
	}
	common.BindFlag(loadCmd, func(f *pflag.FlagSet) {
		f.String("path", "", "module path")
		f.String("modules", "", "modules list,eg: basic,extend")
		f.StringP("bundle", "", "", "bundle name")
		f.String("3rd", "", "build 3rd-party modules")
		f.String("artifact", "", "exist module artifact")
	})
	common.BindFlagCompletions(loadCmd, func(comp carapace.ActionMap) {
		comp["path"] = carapace.ActionFiles()
		comp["modules"] = common.ModulesCompleter()
		comp["artifact"] = common.ModuleArtifactsCompleter(con)
	})
	common.BindArgCompletions(loadCmd, nil,
		carapace.ActionFiles().Usage("path to the module file"))

	unloadCmd := &cobra.Command{
		Use:   "unload [bundle_name]",
		Short: "Unload a module bundle by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UnloadModuleCmd(cmd, con)
		},
	}
	common.BindArgCompletions(unloadCmd, nil,
		common.SessionBundleCompleter(con).Usage("bundle name to unload"))

	refreshCmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh module",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RefreshModuleCmd(cmd, con)
		},
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ClearCmd(cmd, con)
		},
	}

	moduleCmd.AddCommand(listCmd, loadCmd, unloadCmd, refreshCmd, clearCmd)
	return []*cobra.Command{moduleCmd}
}

func Register(con *core.Console) {
	con.RegisterImplantFunc(
		consts.ModuleListModule,
		ListModules,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			resp := ctx.Spite.GetModules()
			sess := con.AddSession(ctx.Session)
			if sess.Data != nil {
				sess.Data.BundleMap = resp.GetBundleMap()
			}
			con.RefreshCmd(sess)
			return resp.Modules, nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			modules := content.Spite.GetModules()
			if len(modules.Modules) == 0 {
				return "No modules found.", nil
			}

			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				table.NewFlexColumn("Module", "Module", 2),
				table.NewFlexColumn("Bundle", "Bundle", 1),
				table.NewFlexColumn("Help", "Help", 3),
			}, true)
			bundleMap := modules.GetBundleMap()
			for _, module := range modules.GetModules() {
				var short string
				if cmd := con.CMDs[module]; cmd != nil {
					short = cmd.Short
				}
				bundle := bundleMap[module]
				row = table.NewRow(
					table.RowData{
						"Module": module,
						"Bundle": bundle,
						"Help":   short,
					})
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetMultiline()
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.RegisterImplantFunc(
		consts.ModuleLoadModule,
		LoadModule,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			resp := ctx.Spite.GetModules()
			ctx.Session.Modules = append(ctx.Session.Modules, resp.Modules...)
			sess := con.AddSession(ctx.Session)
			if sess.Data != nil {
				if sess.Data.BundleMap == nil {
					sess.Data.BundleMap = make(map[string]string)
				}
				for k, v := range resp.GetBundleMap() {
					sess.Data.BundleMap[k] = v
				}
			}
			con.RefreshCmd(sess)
			return resp.Modules, nil
		},
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleLoadModule,
		consts.ModuleLoadModule,
		consts.ModuleLoadModule+"(active(),\"bundle_name\",\"module_file.dll\")",
		[]string{
			"session: special session",
			"bundle_name: bundle name",
			"path: path to the module file",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleUnloadModule,
		unloadModule,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			resp := ctx.Spite.GetModules()
			ctx.Session.Modules = resp.Modules
			sess := con.AddSession(ctx.Session)
			if sess.Data != nil {
				sess.Data.BundleMap = resp.GetBundleMap()
			}
			con.RefreshCmd(sess)
			return resp.Modules, nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			modules := content.Spite.GetModules()
			remaining := modules.GetModules()
			if len(remaining) == 0 {
				return "All modules unloaded.", nil
			}

			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				table.NewFlexColumn("Module", "Module", 2),
				table.NewFlexColumn("Bundle", "Bundle", 1),
				table.NewFlexColumn("Help", "Help", 3),
			}, true)
			bundleMap := modules.GetBundleMap()
			for _, module := range remaining {
				var short string
				if cmd := con.CMDs[module]; cmd != nil {
					short = cmd.Short
				}
				bundle := bundleMap[module]
				row = table.NewRow(
					table.RowData{
						"Module": module,
						"Bundle": bundle,
						"Help":   short,
					})
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetMultiline()
			tableModel.SetRows(rowEntries)
			return "Unloaded successfully. Remaining modules:\n" + tableModel.View(), nil
		})

	con.AddCommandFuncHelper(
		consts.ModuleUnloadModule,
		consts.ModuleUnloadModule,
		consts.ModuleUnloadModule+"(active(),\"bundle_name\")",
		[]string{
			"session: special session",
			"bundle: bundle name to unload",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleRefreshModule,
		refreshModule,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			resp := ctx.Spite.GetModules()
			sess := con.AddSession(ctx.Session)
			if sess.Data != nil {
				sess.Data.BundleMap = resp.GetBundleMap()
			}
			con.RefreshCmd(sess)
			return resp.Modules, nil
		},
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleRefreshModule,
		consts.ModuleRefreshModule,
		consts.ModuleRefreshModule+"(active())",
		[]string{
			"session: special session",
		},
		[]string{"task"})

	// clear
	con.RegisterImplantFunc(
		consts.ModuleClear,
		clearAll,
		"",
		nil,
		output.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleClear,
		consts.ModuleClear,
		consts.ModuleClear+"(active())",
		[]string{
			"session: special session",
		},
		[]string{"task"})

	con.RegisterServerFunc("check_module", func(con *core.Console, sess *client.Session, module string) (bool, error) {
		session, err := con.UpdateSession(sess.SessionId)
		if err != nil {
			return false, err
		}
		return slices.Contains(session.Modules, module), nil
	}, nil)
}
