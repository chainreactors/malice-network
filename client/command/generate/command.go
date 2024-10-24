package generate

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		Args:  cobra.MaximumNArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			ProfileNewCmd(cmd, con)
			return
		},
		Example: `~~~`,
	}
	common.BindArgCompletions(newCmd, nil,
		common.ListenerIDCompleter(con), common.PipelineNameCompleter(con, newCmd),
		carapace.ActionValues().Usage("build target"),
		carapace.ActionValues().Usage("profile name"),
	)
	common.BindFlag(newCmd, func(f *pflag.FlagSet) {
		f.String("type", "", "Set build type")
		f.String("proxy", "", "Set proxy")
		f.String("obfuscate", "", "Set obfuscate")
		f.StringSlice("modules", []string{}, "Set modules e.g.: execute_exe,execute_dll")
		f.String("ca", "", "Set ca")

		f.Int("interval", 10, "Set interval")
		f.Int("jitter", 5, "Set jitter")
	})

	profileCmd.AddCommand(newCmd)

	generateCmd := &cobra.Command{
		Use:     consts.CommandGenerate,
		Short:   "Generate",
		GroupID: consts.GeneratorGroup,
	}

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
	common.BindArgCompletions(peCmd, nil, common.ProfileCompelete(con))

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
	common.BindArgCompletions(moduleCmd, nil, common.ProfileCompelete(con))

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
	common.BindArgCompletions(shellCodeCmd, nil, common.ProfileCompelete(con))

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
	common.BindArgCompletions(stage0Cmd, nil, common.ProfileCompelete(con))

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
	common.BindArgCompletions(stage1Cmd, nil, common.ProfileCompelete(con))

	generateCmd.AddCommand(peCmd, moduleCmd, shellCodeCmd, stage0Cmd, stage1Cmd, shellCodeCmd)

	return []*cobra.Command{profileCmd, generateCmd}
}
