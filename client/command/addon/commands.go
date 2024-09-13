package addon

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	listaddonCmd := &cobra.Command{
		Use:   consts.ModuleListAddon,
		Short: "List all addons",
		Run: func(cmd *cobra.Command, args []string) {
			AddonListCmd(cmd, con)
			return
		},
	}

	loadaddonCmd := &cobra.Command{
		Use:   consts.ModuleLoadAddon,
		Short: "Load an addon",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			LoadAddonCmd(cmd, con)
			return
		},
	}

	common.BindFlag(loadaddonCmd, func(f *pflag.FlagSet) {
		f.StringP("module", "m", "", "module type")
		f.StringP("name", "n", "", "addon name")
		//f.StringP("method", "t", "inline", "method type")
	})

	common.BindArgCompletions(loadaddonCmd, nil,
		carapace.ActionFiles().Usage("path the addon file to load"))

	common.BindFlagCompletions(loadaddonCmd, func(comp carapace.ActionMap) {
		comp["module"] = carapace.ActionValues(consts.ExecuteModules...).Usage("executable module")
		//comp["method"] = carapace.ActionValues("inline", "execute", "shellcode").Usage("method types")
	})

	execAddonCmd := &cobra.Command{
		Use:   consts.ModuleExecuteAddon,
		Short: "Execute an addon",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteAddonCmd(cmd, con)
			return
		},
	}
	common.BindFlag(execAddonCmd, common.SacrificeFlagSet)
	common.BindArgCompletions(execAddonCmd, nil, common.SessionAddonComplete(con))

	return []*cobra.Command{listaddonCmd, loadaddonCmd, execAddonCmd}
}

func Register(con *repl.Console) {
	for name, addon := range loadedAddons {
		intermediate.RegisterInternalFunc(name, addon.Func)
	}
}
