package generate

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	profileCmd := &cobra.Command{
		Use:   consts.CommandProfile,
		Short: "Show all compile profile ",
		Long:  "Show all compile profile",
		Run: func(cmd *cobra.Command, args []string) {
			ProfileShowCmd(cmd, con)
		},
		GroupID: consts.GeneratorGroup,
	}

	newCmd := &cobra.Command{
		Use:   "new",
		Short: "New compile profile",
		Run: func(cmd *cobra.Command, args []string) {
			ProfileNewCmd(cmd, con)
			return
		},
		Example: `~~~`,
	}
	common.BindFlag(newCmd, common.ProfileSet)
	common.BindFlagCompletions(newCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues("profile name")
		comp["target"] = carapace.ActionValues("x86", "x64", "x86_64", "arm", "arm64")
		comp["pipeline_id"] = common.AllPipelineComplete(con)
		comp["type"] = carapace.ActionValues("dll", "exe", "shellcode", "stage0", "stage1")
		comp["proxy"] = carapace.ActionValues("http", "socks5")
		comp["obfuscate"] = carapace.ActionValues("true", "false")
		comp["modules"] = carapace.ActionValues("e.g.: execute_exe,execute_dll")
		comp["ca"] = carapace.ActionValues("true", "false")

		comp["interval"] = carapace.ActionValues("5")
		comp["jitter"] = carapace.ActionValues("0.2")
	})

	profileCmd.AddCommand(newCmd)

	generateCmd := &cobra.Command{
		Use:     consts.CommandGenerate,
		Short:   "Generate",
		GroupID: consts.GeneratorGroup,
	}
	// build beacon --format/-f exe,dll,shellcode -i 1.1.1 -m load_pe
	peCmd := &cobra.Command{
		Use:   consts.CommandPE,
		Short: "Generate PE",
		Long:  "Generate PE",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			PECmd(cmd, con)
		},
	}
	common.BindFlag(peCmd, common.GenerateFlagSet)
	common.BindArgCompletions(peCmd, nil, common.ProfileComplete(con))

	moduleCmd := &cobra.Command{
		Use:   consts.CommandModule,
		Short: "Generate Module DLL",
		Long:  "Generate Module DLL",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ModuleCmd(cmd, con)
		},
	}

	common.BindFlag(moduleCmd, common.GenerateFlagSet)
	common.BindArgCompletions(moduleCmd, nil, common.ProfileComplete(con))

	shellCodeCmd := &cobra.Command{
		Use:   consts.CommandShellCode,
		Short: "Generate ShellCode",
		Long:  "Generate ShellCode",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ShellCodeCmd(cmd, con)
		},
	}

	common.BindFlag(shellCodeCmd, common.GenerateFlagSet)
	common.BindArgCompletions(shellCodeCmd, nil, common.ProfileComplete(con))

	stage0Cmd := &cobra.Command{
		Use:   consts.CommandStage0,
		Short: "Generate Stage0",
		Long:  "Generate Stage0",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			Stage0Cmd(cmd, con)
		},
	}

	common.BindFlag(stage0Cmd, common.GenerateFlagSet)
	common.BindArgCompletions(stage0Cmd, nil, common.ProfileComplete(con))

	stage1Cmd := &cobra.Command{
		Use:   consts.CommandStage1,
		Short: "Generate Stage1",
		Long:  "Generate Stage1",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			Stage1Cmd(cmd, con)
		},
	}

	common.BindFlag(stage1Cmd, common.GenerateFlagSet)
	common.BindArgCompletions(stage1Cmd, nil, common.ProfileComplete(con))

	generateCmd.AddCommand(peCmd, moduleCmd, shellCodeCmd, stage0Cmd, stage1Cmd, shellCodeCmd)

	return []*cobra.Command{profileCmd, generateCmd}
}
