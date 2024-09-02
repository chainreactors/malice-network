package sys

import (
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Commands(con *console.Console) []*cobra.Command {
	whoamiCmd := &cobra.Command{
		Use:   consts.ModuleWhoami,
		Short: "Print current user",
		Long:  help.GetHelpFor(consts.ModuleWhoami),
		Run: func(cmd *cobra.Command, args []string) {
			WhoamiCmd(cmd, con)
			return
		},
	}

	killCmd := &cobra.Command{
		Use:   consts.ModuleKill,
		Short: "Kill the process",
		Long:  help.GetHelpFor(consts.ModuleKill),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			KillCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(killCmd).PositionalCompletion(
		carapace.ActionValues().Usage("process pid"),
	)

	psCmd := &cobra.Command{
		Use:   consts.ModulePs,
		Short: "List processes",
		Long:  help.GetHelpFor(consts.ModulePs),
		Run: func(cmd *cobra.Command, args []string) {
			PsCmd(cmd, con)
			return
		},
	}

	envCmd := &cobra.Command{
		Use:   consts.ModuleEnv,
		Short: "List environment variables",
		Long:  help.GetHelpFor(consts.ModuleEnv),
		Run: func(cmd *cobra.Command, args []string) {
			EnvCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	setEnvCmd := &cobra.Command{
		Use:   consts.ModuleSetEnv,
		Short: "Set environment variable",
		Long:  help.GetHelpFor(consts.ModuleSetEnv),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			SetEnvCmd(cmd, con)
			return
		},
	}

	carapace.Gen(setEnvCmd).PositionalCompletion(
		carapace.ActionValues().Usage("environment variable"),
		carapace.ActionValues().Usage("value"),
	)

	unSetEnvCmd := &cobra.Command{
		Use:   consts.ModuleUnsetEnv,
		Short: "Unset environment variable",
		Long:  help.GetHelpFor(consts.ModuleUnsetEnv),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UnsetEnvCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(unSetEnvCmd).PositionalCompletion(
		carapace.ActionValues().Usage("environment variable"),
	)

	netstatCmd := &cobra.Command{
		Use:   consts.ModuleNetstat,
		Short: "List network connections",
		Long:  help.GetHelpFor(consts.ModuleNetstat),
		Run: func(cmd *cobra.Command, args []string) {
			NetstatCmd(cmd, con)
			return
		},
	}

	infoCmd := &cobra.Command{
		Use:   consts.ModuleInfo,
		Short: "get basic sys info",
		Long:  help.GetHelpFor(consts.ModuleInfo),
		Run: func(cmd *cobra.Command, args []string) {
			InfoCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	return []*cobra.Command{
		whoamiCmd,
		killCmd,
		psCmd,
		envCmd,
		setEnvCmd,
		unSetEnvCmd,
		netstatCmd,
		infoCmd,
	}
}
