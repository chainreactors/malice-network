package saas

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	saasCmd := &cobra.Command{
		Use:   consts.CommandSaas,
		Short: "saas",
	}
	beaconCmd := &cobra.Command{
		Use:   consts.CommandBuildBeacon,
		Short: "Build a beacon by saas",
		Long: `Generate a beacon artifact by saas based on the specified profile.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return BeaconCmd(cmd, con)
		},
		Example: `~~~
// Build a beacon by saas
saas beacon --target x86_64-unknown-linux-musl --profile beacon_profile

// Build a beacon using additional modules by saas
saas beacon --target x86_64-pc-windows-msvc --profile beacon_profile --modules full

// Build a beacon using SRDI technology by saas
saas beacon --target x86_64-pc-windows-msvc --profile beacon_profile --srdi

~~~`,
	}
	common.BindFlag(beaconCmd, common.GenerateFlagSet)
	beaconCmd.MarkFlagRequired("target")
	beaconCmd.MarkFlagRequired("profile")
	common.BindFlagCompletions(beaconCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
	})

	bindCmd := &cobra.Command{
		Use:   consts.CommandBuildBind,
		Short: "Build a bind payload by saas",
		Long:  `Generate a bind payload by saas that connects a client to the server.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return BindCmd(cmd, con)
		},
		Example: `~~~
// Build a bind payload by saas
saas bind --target x86_64-pc-windows-msvc --profile bind_profile

// Build a bind payload with additional modules by saas
saas bind --target x86_64-unknown-linux-musl --profile bind_profile --modules base,sys_full

// Build a bind payload with SRDI technology by saas
saas bind --target x86_64-pc-windows-msvc --profile bind_profile --srdi

~~~`,
	}

	common.BindFlag(bindCmd, common.GenerateFlagSet)
	bindCmd.MarkFlagRequired("target")
	bindCmd.MarkFlagRequired("profile")
	common.BindFlagCompletions(bindCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
	})

	preludeCmd := &cobra.Command{
		Use:   consts.CommandBuildPrelude,
		Short: "Build a prelude payload by saas",
		Long: `Generate a prelude payload by saas as part of a multi-stage deployment.
	`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PreludeCmd(cmd, con)
		},
		Example: `~~~
	// Build a prelude payload by saas
	saas prelude --target x86_64-unknown-linux-musl --profile prelude_profile --autorun /path/to/autorun.yaml
	
	// Build a prelude payload with additional modules by saas
	saas prelude --target x86_64-pc-windows-msvc --profile prelude_profile --autorun /path/to/autorun.yaml --modules base,sys_full
	
	// Build a prelude payload with SRDI technology by saas
saas prelude --target x86_64-pc-windows-msvc --profile prelude_profile --autorun /path/to/autorun.yaml --srdi
	~~~`,
	}

	common.BindFlag(preludeCmd, common.GenerateFlagSet, func(f *pflag.FlagSet) {
		f.String("autorun", "", "set autorun.yaml")
	})
	preludeCmd.MarkFlagRequired("target")
	preludeCmd.MarkFlagRequired("profile")
	preludeCmd.MarkFlagRequired("autorun")
	common.BindFlagCompletions(preludeCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
		comp["autorun"] = carapace.ActionFiles().Usage("autorun.yaml path")
	})
	common.BindArgCompletions(preludeCmd, nil, common.ProfileCompleter(con))

	modulesCmd := &cobra.Command{
		Use:   consts.CommandBuildModules,
		Short: "Compile specified modules into DLLs by saas",
		Long: `Compile the specified modules into DLL files for deployment or integration by saas.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ModulesCmd(cmd, con)
		},
		Example: `~~~
// Compile all modules for the Windows platform by saas
saas modules --target x86_64-unknown-linux-musl --profile module_profile

// Compile a predefined feature set of modules (nano) by saas
saas modules --target x86_64-unknown-linux-musl --profile module_profile --modules nano

// Compile specific modules into DLLs by saas
saas modules --target x86_64-pc-windows-msvc --profile module_profile --modules base,execute_dll

// Compile modules with srdi
saas modules --target x86_64-pc-windows-msvc --profile module_profile --srdi
~~~`,
	}
	common.BindFlag(modulesCmd, common.GenerateFlagSet)

	common.BindFlagCompletions(modulesCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
	})

	modulesCmd.MarkFlagRequired("target")
	modulesCmd.MarkFlagRequired("profile")

	pulseCmd := &cobra.Command{
		Use:   consts.CommandBuildPulse,
		Short: "stage 0 shellcode generate by saas",
		Long: `Generate 'pulse' payload by saas,a minimized shellcode template, corresponding to CS artifact, very suitable for loading by various loaders
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return PulseCmd(cmd, con)
		},
		Example: `
~~~
// Build a pulse payload by saas
saas pulse --target x86_64-unknown-linux-musl --profile pulse_profile

// Build a pulse payload with additional modules by saas
saas pulse --target x86_64-pc-windows-msvc --profile pulse_profile --modules base,sys_full
	
// Build a pulse payload with SRDI technology by saas
saas pulse --target x86_64-pc-windows-msvc --profile pulse_profile --srdi

// Build a pulse payload by specifying artifact by saas
saas pulse --target x86_64-pc-windows-msvc --profile pulse_profile --artifact-id 1
~~~
`,
	}
	common.BindFlag(pulseCmd, func(f *pflag.FlagSet) {
		f.String("profile", "", "profile name")
		f.StringP("address", "a", "", "implant address")
		f.String("srdi", "", "enable srdi")
		f.String("target", "", "build target")
		f.Uint32("artifact-id", 0, "load remote shellcode build-id")
	})
	pulseCmd.MarkFlagRequired("target")
	pulseCmd.MarkFlagRequired("profile")
	common.BindFlagCompletions(pulseCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
	})
	saasCmd.AddCommand(beaconCmd, bindCmd, modulesCmd, pulseCmd, preludeCmd)
	return []*cobra.Command{
		saasCmd,
	}
}
