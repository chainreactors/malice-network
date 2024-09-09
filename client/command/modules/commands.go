package modules

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/handler"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
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
		Use:   consts.ModuleLoadModule,
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

	con.RegisterImplantFunc(
		"blist_module",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, fileName string) (*clientpb.Task, error) {
			return ListModules(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			err := handler.HandleMaleficError(ctx.Spite)
			if err != nil {
				return "", err
			}
			resp := ctx.Spite.GetModules()
			var modules []string
			for module := range resp.GetModules() {
				modules = append(modules, fmt.Sprintf("%s", module))
			}
			return strings.Join(modules, ","), nil
		})

	con.RegisterImplantFunc(
		"bload_module",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, bundle string, path string) (*clientpb.Task, error) {
			return LoadModule(rpc, sess, bundle, path)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterImplantFunc(
		"brefresh_module",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session) (*clientpb.Task, error) {
			return refreshModule(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	//clear
	con.RegisterImplantFunc(
		"bclear",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session) (*clientpb.Task, error) {
			return clearAll(rpc, sess)
		},
		func(context *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(context.Spite)
		})
	return []*cobra.Command{
		listModuleCmd,
		loadModuleCmd,
		refreshModuleCmd,
		clearCmd,
	}
}
