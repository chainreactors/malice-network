package action

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/config"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	actionCmd := &cobra.Command{
		Use:   consts.CommandAction,
		Short: "Github action build",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	beaconCmd := &cobra.Command{
		Use:   consts.CommandBuildBeacon,
		Short: "run github action to build beacon",
		Long:  `Generate a beacon artifact based on the specified profile by github workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunBeaconWorkFlowCmd(cmd, con)
		},
		Example: `~~~
// Build a beacon by workflow
action beacon --target x86_64-unknown-linux-musl --profile beacon_profile

// Build a beacon using additional modules by workflow
action beacon --target x86_64-pc-windows-msvc --profile beacon_profile --modules full

~~~`,
	}

	common.BindFlag(beaconCmd, config.GithubFlagSet, common.GenerateFlagSet)
	common.BindFlagCompletions(beaconCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	beaconCmd.MarkFlagRequired("target")
	beaconCmd.MarkFlagRequired("profile")

	bindCmd := &cobra.Command{
		Use:   consts.CommandBuildBind,
		Short: "run github action to build bind",
		Long:  `Generate a bind payload that connects a client to the server by github workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunBindWorkFlowCmd(cmd, con)
		},
		Example: `~~~
// Build a bind payload by github workflow
action bind --target x86_64-pc-windows-msvc --profile bind_profile

// Build a bind payload with additional modules by github workflow
action bind --target x86_64-unknown-linux-musl --profile bind_profile --modules base,sys_full

~~~`,
	}

	common.BindFlag(bindCmd, config.GithubFlagSet, common.GenerateFlagSet)
	common.BindFlagCompletions(bindCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	bindCmd.MarkFlagRequired("target")
	bindCmd.MarkFlagRequired("profile")

	modulesCmd := &cobra.Command{
		Use:   consts.CommandBuildModules,
		Short: "run github action to build modules",
		Long: `Compile the specified modules into DLL files for deployment or integration by github workflow.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunModulesWorkFlowCmd(cmd, con)
		},
		Example: `~~~
// Compile all modules for the Windows platform by github workflow
action modules --target x86_64-unknown-linux-musl --profile module_profile

// Compile a predefined feature set of modules (nano) by github workflow
action modules --target x86_64-unknown-linux-musl --profile module_profile --modules nano

// Compile specific modules into DLLs by github workflow
action modules --target x86_64-pc-windows-msvc --profile module_profile --modules base,execute_dll
~~~`,
	}

	common.BindFlag(modulesCmd, config.GithubFlagSet, common.GenerateFlagSet)
	common.BindFlagCompletions(modulesCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	modulesCmd.MarkFlagRequired("target")
	modulesCmd.MarkFlagRequired("profile")

	pulseCmd := &cobra.Command{
		Use:   consts.CommandBuildPulse,
		Short: "run github action to build pulse",
		Long: `Generate 'pulse' payload,a minimized shellcode template, corresponding to CS artifact, very suitable for loading by various loaders by github workflow.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunPulseWorkFlowCmd(cmd, con)
		},
		Example: `
~~~ 
// Build a pulse payload by github workflow
action pulse --target x86_64-unknown-linux-musl --profile pulse_profile

// Build a pulse payload with additional modules by github workflow
action pulse --target x86_64-pc-windows-msvc --profile pulse_profile --modules base,sys_full

// Build a pulse payload by specifying artifact by github workflow
action pulse --target x86_64-pc-windows-msvc --profile pulse_profile --artifact-id 1
~~~
`,
	}

	common.BindFlag(pulseCmd, config.GithubFlagSet, common.GenerateFlagSet, func(f *pflag.FlagSet) {
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
		Long: `Generate a prelude payload as part of a multi-stage deployment by github workflow.
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunPreludeWorkFlowCmd(cmd, con)
		},
		Example: `~~~
	// Build a prelude payload by github workflow
	action prelude --target x86_64-unknown-linux-musl --profile prelude_profile --autorun /path/to/autorun.yaml
	
	// Build a prelude payload with additional modules by github workflow
	action prelude --target x86_64-pc-windows-msvc --profile prelude_profile --autorun /path/to/autorun.yaml --modules base,sys_full
	~~~`,
	}

	common.BindFlag(preludeCmd, config.GithubFlagSet, common.GenerateFlagSet, func(f *pflag.FlagSet) {
		f.String("autorun", "", "autorun.yaml path")
	})
	common.BindFlagCompletions(preludeCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
	})
	preludeCmd.MarkFlagRequired("target")
	preludeCmd.MarkFlagRequired("profile")
	preludeCmd.MarkFlagRequired("autorun")

	actionCmd.AddCommand(beaconCmd, bindCmd, preludeCmd, pulseCmd, modulesCmd)
	return []*cobra.Command{actionCmd}
}

func Register(con *repl.Console) {
	settings, err := assets.GetSetting()
	if err != nil {
		con.Log.Errorf("Get settings failed: %v", err)
		return
	}
	con.RegisterServerFunc(consts.CommandAction+"_"+consts.CommandActionRun, func(con *repl.Console, msg string) (*clientpb.Builder, error) {
		return RunWorkFlow(con, &clientpb.GithubWorkflowRequest{
			Owner:      settings.GithubOwner,
			Repo:       settings.GithubRepo,
			Token:      settings.GithubToken,
			WorkflowId: settings.GithubWorkflowFile,
		})
	}, nil)
}
