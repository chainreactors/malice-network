package action

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	actionCmd := &cobra.Command{
		Use:   consts.CommandAction,
		Short: "Github action",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	runCmd := &cobra.Command{
		Use:   consts.CommandActionRun,
		Short: " run github workflow",
	}

	beaconCmd := &cobra.Command{
		Use:   consts.CommandBuildBeacon,
		Short: "run github action to build beacon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunBeaconWorkFlowCmd(cmd, con)
		},
	}

	common.BindFlag(beaconCmd, common.GithubFlagSet, common.GenerateFlagSet)
	common.BindFlagCompletions(beaconCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	beaconCmd.MarkFlagRequired("target")
	beaconCmd.MarkFlagRequired("profile")

	bindCmd := &cobra.Command{
		Use:   consts.CommandBuildBind,
		Short: "run github action to build bind",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunBindWorkFlowCmd(cmd, con)
		},
	}

	common.BindFlag(bindCmd, common.GithubFlagSet, common.GenerateFlagSet)
	common.BindFlagCompletions(bindCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	bindCmd.MarkFlagRequired("target")
	bindCmd.MarkFlagRequired("profile")

	modulesCmd := &cobra.Command{
		Use:   consts.CommandBuildModules,
		Short: "run github action to build modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunModulesWorkFlowCmd(cmd, con)
		},
	}

	common.BindFlag(modulesCmd, common.GithubFlagSet, common.GenerateFlagSet)
	common.BindFlagCompletions(modulesCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	modulesCmd.MarkFlagRequired("target")
	modulesCmd.MarkFlagRequired("profile")

	pulseCmd := &cobra.Command{
		Use:   consts.CommandBuildPulse,
		Short: "run github action to build pulse",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunPulseWorkFlowCmd(cmd, con)
		},
	}

	common.BindFlag(pulseCmd, common.GithubFlagSet, common.GenerateFlagSet, func(f *pflag.FlagSet) {
		f.Uint32("artifact-id", 0, "load remote shellcode build-id")
	})
	common.BindFlagCompletions(pulseCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	pulseCmd.MarkFlagRequired("target")
	pulseCmd.MarkFlagRequired("profile")

	preludeCmd := &cobra.Command{
		Use:   consts.CommandBuildPrelude,
		Short: "run github action to build prelude",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunPreludeWorkFlowCmd(cmd, con)
		},
	}

	common.BindFlag(preludeCmd, common.GithubFlagSet, common.GenerateFlagSet, func(f *pflag.FlagSet) {
		f.String("autorun", "", "autorun.yaml path")
	})
	common.BindFlagCompletions(preludeCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	preludeCmd.MarkFlagRequired("target")
	preludeCmd.MarkFlagRequired("profile")
	preludeCmd.MarkFlagRequired("autorun")

	runCmd.AddCommand(beaconCmd, bindCmd, preludeCmd, pulseCmd, modulesCmd)
	actionCmd.AddCommand(runCmd)
	return []*cobra.Command{actionCmd}
}

func Register(con *repl.Console) {
	settings := assets.GetProfile().Settings
	con.RegisterServerFunc(consts.CommandAction+"_"+consts.CommandActionRun, func(con *repl.Console, msg string) (*clientpb.Builder, error) {
		return RunWorkFlow(con, &clientpb.WorkflowRequest{
			Owner:      settings.GithubOwner,
			Repo:       settings.GithubRepo,
			Token:      settings.GithubToken,
			WorkflowId: settings.GithubWorkflowFile,
		})
	}, nil)
}
