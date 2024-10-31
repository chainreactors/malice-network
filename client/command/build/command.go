package build

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
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProfileShowCmd(cmd, con)
		},
		GroupID: consts.GeneratorGroup,
	}

	newCmd := &cobra.Command{
		Use:   "new",
		Short: "New compile profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProfileNewCmd(cmd, con)
		},
		Example: `~~~`,
	}
	common.BindFlag(newCmd, common.ProfileSet)
	common.BindFlagCompletions(newCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues("profile name")
		comp["target"] = common.TargetComplete(con)
		comp["pipeline_id"] = common.AllPipelineComplete(con)
		comp["type"] = common.TypeComplete(con)
		comp["proxy"] = carapace.ActionValues("http", "socks5")
		comp["obfuscate"] = carapace.ActionValues("true", "false")
		comp["modules"] = carapace.ActionValues("e.g.: execute_exe,execute_dll")
		comp["ca"] = carapace.ActionValues("true", "false")

		comp["interval"] = carapace.ActionValues("5")
		comp["jitter"] = carapace.ActionValues("0.2")
	})

	profileCmd.AddCommand(newCmd)

	buildCmd := &cobra.Command{
		Use:     consts.CommandBuild,
		Short:   "build",
		GroupID: consts.GeneratorGroup,
	}
	// build beacon --format/-f exe,dll,shellcode -i 1.1.1 -m load_pe
	beaconCmd := &cobra.Command{
		Use:   consts.CommandBeacon,
		Short: "build beacon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return BeaconCmd(cmd, con)
		},
	}
	common.BindFlag(beaconCmd, common.GenerateFlagSet)
	common.BindFlagCompletions(beaconCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileComplete(con)
		comp["target"] = common.TargetComplete(con)
		comp["format"] = common.TypeComplete(con)
	})
	common.BindArgCompletions(beaconCmd, nil, common.ProfileComplete(con))

	bindCmd := &cobra.Command{
		Use:   consts.CommandBind,
		Short: "build bind",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return BindCmd(cmd, con)
		},
	}

	common.BindFlag(bindCmd, common.GenerateFlagSet)
	common.BindFlagCompletions(bindCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileComplete(con)
		comp["target"] = common.TargetComplete(con)
		comp["format"] = common.TypeComplete(con)
	})
	common.BindArgCompletions(bindCmd, nil, common.ProfileComplete(con))

	shellCodeCmd := &cobra.Command{
		Use:   consts.CommandShellCode,
		Short: "build ShellCode",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ShellCodeCmd(cmd, con)
		},
	}

	common.BindFlag(shellCodeCmd, common.GenerateFlagSet)
	common.BindFlagCompletions(shellCodeCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileComplete(con)
		comp["target"] = common.TargetComplete(con)
		comp["format"] = common.TypeComplete(con)
	})
	common.BindArgCompletions(shellCodeCmd, nil, common.ProfileComplete(con))

	preludeCmd := &cobra.Command{
		Use:   consts.CommandPrelude,
		Short: "build prelude",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PreludeCmd(cmd, con)
		},
	}

	common.BindFlag(preludeCmd, common.GenerateFlagSet)
	common.BindFlagCompletions(preludeCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileComplete(con)
		comp["target"] = common.TargetComplete(con)
		comp["format"] = common.TypeComplete(con)
	})
	common.BindArgCompletions(preludeCmd, nil, common.ProfileComplete(con))

	downloadCmd := &cobra.Command{
		Use:   consts.CommandDownload,
		Short: "download build output file in server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return DownloadCmd(cmd, con)
		},
	}
	buildCmd.AddCommand(beaconCmd, bindCmd, shellCodeCmd, preludeCmd, downloadCmd)

	return []*cobra.Command{profileCmd, buildCmd}
}
