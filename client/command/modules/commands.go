package modules

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"
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
		// Long:  help.FormatLongHelp(consts.ModuleLoadModule),duan
		RunE: func(cmd *cobra.Command, args []string) error {
			return LoadModuleCmd(cmd, con)
		},
		Example: `load module from malefic-modules
before loading, you can list the current modules: 
~~~
execute_addon,exec ...
~~~
then you can load module
~~~
load_module --path <module_file.dll>
~~~
you can see more modules loaded by list_module
~~~
execute_addon,clear,ps,powershell...
~~~
`}

	common.BindFlag(loadModuleCmd, func(f *pflag.FlagSet) {
		f.String("path", "", "module path")
		f.String("modules", "", "modules list,eg: basic,extend")
		f.StringP("bundle", "", "", "bundle name")
		f.String("3rd", "", "build 3rd-party modules")
		f.String("artifact", "", "exist module artifact")
	})
	common.BindFlagCompletions(loadModuleCmd, func(comp carapace.ActionMap) {
		comp["path"] = carapace.ActionFiles()
		comp["modules"] = common.ModulesCompleter()
		comp["artifact"] = common.ModuleArtifactsCompleter(con)
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
			con.RefreshCmd(con.AddSession(ctx.Session))
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
				table.NewColumn("Module", "Module", 20),
				table.NewColumn("Help", "Help", 30),
			}, true)
			for _, module := range modules.GetModules() {
				row = table.NewRow(
					table.RowData{
						"Module": module,
						"Help":   "",
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
			con.RefreshCmd(con.AddSession(ctx.Session))
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
		consts.ModuleRefreshModule,
		refreshModule,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			resp := ctx.Spite.GetModules()
			con.RefreshCmd(con.AddSession(ctx.Session))
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

	//clear
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

	con.RegisterServerFunc("check_module", func(con *repl.Console, sess *core.Session, module string) (bool, error) {
		session, err := con.UpdateSession(sess.SessionId)
		if err != nil {
			return false, err
		}
		return slices.Contains(session.Modules, module), nil
	}, nil)
}
