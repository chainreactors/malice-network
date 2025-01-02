package build

import (
	"fmt"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	profileCmd := &cobra.Command{
		Use:   consts.CommandProfile,
		Short: "compile profile ",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	listCmd := &cobra.Command{
		Use:   consts.CommandProfileList,
		Short: "List all compile profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProfileShowCmd(cmd, con)
		},
		Example: `~~~
// List all compile profiles
profile list
~~~`,
	}

	loadProfileCmd := &cobra.Command{
		Use:   consts.CommandProfileLoad,
		Short: "Load exist implant profile",
		Long: `
The **profile load** command requires a valid configuration file path (e.g., **config.yaml**) to load settings. This file specifies attributes necessary for generating the compile profile.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProfileLoadCmd(cmd, con)
		},
		Example: `~~~
// Create a new profile using network configuration in pipeline
profile load /path/to/config.yaml --name my_profile --pipeline pipeline_name

// Create a profile with specific modules
profile load /path/to/config.yaml --name my_profile --modules base,sys_full --pipeline pipeline_name

// Create a profile with custom interval and jitter
profile load /path/to/config.yaml --name my_profile --interval 10 --jitter 0.3 --pipeline pipeline_name

// Create a profile for pulse
profile load /path/to/config.yaml --name my_profile --pipeline pipeline_name --pulse-pipeline pulse_pipeline_name
~~~`,
	}
	common.BindFlag(loadProfileCmd, common.ProfileSet)
	loadProfileCmd.MarkFlagRequired("pipeline")
	loadProfileCmd.MarkFlagRequired("name")
	common.BindFlagCompletions(loadProfileCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues("profile name")
		//comp["target"] = common.BuildTargetCompleter(con)
		comp["pipeline"] = common.AllPipelineCompleter(con)
		comp["pulse-pipeline"] = common.AllPipelineCompleter(con)
		//comp["proxy"] = carapace.ActionValues("").Usage("")
		//comp["obfuscate"] = carapace.ActionValues("true", "false")
		comp["modules"] = carapace.ActionValues("e.g.: execute_exe,execute_dll")
		comp["ca"] = carapace.ActionValues("true", "false")

		comp["interval"] = carapace.ActionValues("5")
		comp["jitter"] = carapace.ActionValues("0.2")
	})
	common.BindArgCompletions(loadProfileCmd, nil, carapace.ActionFiles().Usage("profile path"))

	newProfileCmd := &cobra.Command{
		Use:   consts.CommandProfileNew,
		Short: "Create new compile profile with default profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProfileNewCmd(cmd, con)
		},
		Example: `
~~~
profile new --name my_profile --pipeline default_tcp
~~~
`,
	}
	common.BindFlag(newProfileCmd, common.ProfileSet)
	newProfileCmd.MarkFlagRequired("pipeline")
	newProfileCmd.MarkFlagRequired("name")
	common.BindFlagCompletions(newProfileCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues("profile name")
		comp["pipeline"] = common.AllPipelineCompleter(con)
		comp["pulse-pipeline"] = common.AllPipelineCompleter(con)
	})

	deleteProfileCmd := &cobra.Command{
		Use:   consts.CommandProfileDelete,
		Short: "Delete a compile profile in server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProfileDeleteCmd(cmd, con)
		},
		Example: `
~~~
profile delete --name profile_name
~~~
`,
	}
	common.BindArgCompletions(deleteProfileCmd, nil,
		common.ProfileCompleter(con))

	profileCmd.AddCommand(listCmd, loadProfileCmd, newProfileCmd, deleteProfileCmd)

	buildCmd := &cobra.Command{
		Use:   consts.CommandBuild,
		Short: "build",
	}
	// build beacon --format/-f exe,dll,shellcode -i 1.1.1 -m load_pe
	beaconCmd := &cobra.Command{
		Use:   consts.CommandBuildBeacon,
		Short: "Build a beacon",
		Long: `Generate a beacon artifact based on the specified profile.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return BeaconCmd(cmd, con)
		},
		Example: `~~~
// Build a beacon
build beacon --target x86_64-unknown-linux-musl --profile beacon_profile

// Build a beacon using additional modules
build beacon --target x86_64-pc-windows-msvc --profile beacon_profile --modules full

// Build a beacon using SRDI technology
build beacon --target x86_64-pc-windows-msvc --profile beacon_profile --srdi

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
		Short: "Build a bind payload",
		Long:  `Generate a bind payload that connects a client to the server.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return BindCmd(cmd, con)
		},
		Example: `~~~
// Build a bind payload
build bind --target x86_64-pc-windows-msvc --profile bind_profile

// Build a bind payload with additional modules
build bind --target x86_64-unknown-linux-musl --profile bind_profile --modules base,sys_full

// Build a bind payload with SRDI technology
build bind --target x86_64-pc-windows-msvc --profile bind_profile --srdi

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
		Short: "Build a prelude payload",
		Long: `Generate a prelude payload as part of a multi-stage deployment.
	`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PreludeCmd(cmd, con)
		},
		Example: `~~~
	// Build a prelude payload
	build prelude --target x86_64-unknown-linux-musl --profile prelude_profile --autorun /path/to/autorun.yaml
	
	// Build a prelude payload with additional modules
	build prelude --target x86_64-pc-windows-msvc --profile prelude_profile --autorun /path/to/autorun.yaml --modules base,sys_full
	
	// Build a prelude payload with SRDI technology
	build prelude --target x86_64-pc-windows-msvc --profile prelude_profile --autorun /path/to/autorun.yaml --srdi
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
		Short: "Compile specified modules into DLLs",
		Long: `Compile the specified modules into DLL files for deployment or integration.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ModulesCmd(cmd, con)
		},
		Example: `~~~
// Compile all modules for the Windows platform
build modules --target x86_64-unknown-linux-musl --profile module_profile

// Compile a predefined feature set of modules (nano)
build modules --target x86_64-unknown-linux-musl --profile module_profile --modules nano

// Compile specific modules into DLLs
build modules --target x86_64-pc-windows-msvc --profile module_profile --modules base,execute_dll

// Compile modules with srdi
build modules --target x86_64-pc-windows-msvc --profile module_profile --srdi
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
		Short: "stage 0 shellcode generate",
		Long: `Generate 'pulse' payload,a minimized shellcode template, corresponding to CS artifact, very suitable for loading by various loaders
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return PulseCmd(cmd, con)
		},
		Example: `
~~~
// Build a pulse payload
build pulse --target x86_64-unknown-linux-musl --profile pulse_profile

// Build a pulse payload with additional modules
build pulse --target x86_64-pc-windows-msvc --profile pulse_profile --modules base,sys_full
	
// Build a pulse payload with SRDI technology
build pulse --target x86_64-pc-windows-msvc --profile pulse_profile --srdi

// Build a pulse payload by specifying artifact
build pulse --target x86_64-pc-windows-msvc --profile pulse_profile --artifact-id 1
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

	logCmd := &cobra.Command{
		Use:   consts.CommandBuildLog,
		Short: "Show build log",
		Long:  `Displays the log for the specified number of rows`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return BuildLogCmd(cmd, con)
		},
		Example: `
~~~
build log builder_name --limit 70
~~~
`,
	}
	common.BindFlag(logCmd, func(f *pflag.FlagSet) {
		f.Int("limit", 50, "limit of rows")
	})
	common.BindArgCompletions(logCmd, nil, common.ArtifactNameCompleter(con))

	buildCmd.AddCommand(beaconCmd, bindCmd, modulesCmd, pulseCmd, preludeCmd, logCmd)

	artifactCmd := &cobra.Command{
		Use:   consts.CommandArtifact,
		Short: "artifact manage",
		Long:  "Manage build output files on the server. Use the **list** command to view all available artifacts, **download** to retrieve a specific artifact, and **upload** to add a new artifact to the server.",
	}

	listArtifactCmd := &cobra.Command{
		Use:   consts.CommandArtifactList,
		Short: "list build output file in server",
		Long: `Retrieve a list of all build output files currently stored on the server.

This command fetches metadata about artifacts, such as their names, IDs, and associated build configurations. The artifacts are displayed in a table format for easy navigation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListArtifactCmd(cmd, con)
		},
		Example: `~~~
// List all available build artifacts on the server
artifact list

// Navigate the artifact table and press enter to download a specific artifact
~~~`,
	}

	downloadCmd := &cobra.Command{
		Use:   consts.CommandArtifactDownload,
		Short: "Download a build output file from the server",
		Long: `Download a specific build output file from the server by specifying its unique artifact name.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DownloadArtifactCmd(cmd, con)
		},
		Example: `
// Download a artifact
	artifact download artifact_name

// Download a artifact to specific path
	artifact download artifact_name -o /path/to/output

// Download a shellcode artifact by enabling the 'srdi' flag
	artifact download artifact_name -s
`,
	}
	common.BindFlag(downloadCmd, func(f *pflag.FlagSet) {
		f.StringP("output", "o", "", "output path")
		f.BoolP("srdi", "s", false, "Set to true to download shellcode.")
	})
	common.BindArgCompletions(downloadCmd, nil, common.ArtifactNameCompleter(con))

	uploadCmd := &cobra.Command{
		Use:   consts.CommandArtifactUpload,
		Short: "Upload a build output file to the server",
		Long: `Upload a custom artifact to the server for storage or further use.

`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UploadArtifactCmd(cmd, con)
		},
		Example: `~~~
// Upload an artifact with default settings
artifact upload /path/to/artifact

// Upload an artifact with a specific stage and alias name
artifact upload /path/to/artifact --stage production --name my_artifact

// Upload an artifact and specify its type
artifact upload /path/to/artifact --type DLL
~~~`,
	}
	common.BindArgCompletions(uploadCmd, nil, carapace.ActionFiles().Usage("custom artifact"))
	common.BindFlag(uploadCmd, func(f *pflag.FlagSet) {
		f.StringP("stage", "s", "", "Set stage")
		f.StringP("type", "t", "", "Set type")
		f.StringP("name", "n", "", "alias name")
	})

	deleteCommand := &cobra.Command{
		Use:   consts.CommandArtifactDelete,
		Short: "Delete a artifact file in the server",
		Long: `Delete a specify artifact in the server.

`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteArtifactCmd(cmd, con)
		},
		Example: `
~~~
artifact delete --name artifact_name
~~~
`}

	common.BindArgCompletions(deleteCommand, nil,
		common.ArtifactCompleter(con))

	artifactCmd.AddCommand(listArtifactCmd, downloadCmd, uploadCmd, deleteCommand)

	return []*cobra.Command{profileCmd, buildCmd, artifactCmd}
}

func Register(con *repl.Console) {
	con.RegisterServerFunc("search_artifact", SearchArtifact, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "search build artifact with arch,os,typ and pipeline id",
		Input: []string{
			"pipeline: pipeline id",
			"type: build type, beacon,bind,prelude",
			"format: only support shellcode",
			"arch: arch",
			"os: os",
		},
		Output: []string{
			"builder",
		},
		Example: `search_artifact("x64","windows","beacon","tcp_default", true)`,
	})

	con.RegisterServerFunc("get_artifact", func(con *repl.Console, sess *core.Session, format string) (*clientpb.Artifact, error) {
		artifact := &clientpb.Artifact{Name: sess.Name}
		switch format {
		case "bin", "raw", "shellcode":
			artifact.IsSrdi = true
		}
		artifact, err := con.Rpc.FindArtifact(sess.Context(), artifact)
		if err != nil {
			return nil, err
		}
		return artifact, nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "get artifact with session self",
		Input: []string{
			"sess: session",
			"format: only support shellcode",
		},
		Output: []string{
			"builder",
		},
	})

	con.RegisterServerFunc("upload_artifact", UploadArtifact, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "upload local bin to server build",
	})

	con.RegisterServerFunc("download_artifact", DownloadArtifact, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "download artifact with special build id",
	})

	con.RegisterServerFunc("delete_artifact", DeleteArtifact, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "delete artifact with special build name",
	})

	con.RegisterServerFunc("self_stager", func(con *repl.Console, sess *core.Session) (string, error) {
		artifact, err := SearchArtifact(con, sess.PipelineId, "pulse", "shellcode", sess.Os.Name, sess.Os.Arch)
		if err != nil {
			return "", err
		}
		return string(artifact.Bin), nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "get self artifact stager shellcode",
		Input: []string{
			"sess: session",
		},
		Output: []string{
			"artifact_bin",
		},
		Example: `self_payload(active())`,
	})

	con.RegisterServerFunc("artifact_stager", func(con *repl.Console, pipeline, format, os, arch string) (string, error) {
		artifact, err := SearchArtifact(con, pipeline, "pulse", "shellcode", os, arch)
		if err != nil {
			return "", err
		}
		return string(artifact.Bin), nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "get artifact stager shellcode",
		Input: []string{
			"pipeline: pipeline id",
			"format: reserved parameter",
			"os: os, windows only",
			"arch: arch, x64/x86",
		},
		Output: []string{
			"artifact_bin",
		},
		Example: `artifact_stager("tcp_default","raw","windows","x64")`,
	})
	con.RegisterServerFunc("self_payload", func(con *repl.Console, sess *core.Session) (string, error) {
		artifact, err := SearchArtifact(con, sess.PipelineId, "beacon", "shellcode", sess.Os.Name, sess.Os.Arch)
		if err != nil {
			return "", fmt.Errorf("get artifact error: %s", err)
		}
		return string(artifact.Bin), nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "get self artifact stageless shellcode",
		Input: []string{
			"sess: Session",
		},
		Output: []string{
			"artifact_bin",
		},
		Example: `self_payload(active())`,
	})

	con.RegisterServerFunc("artifact_payload", func(con *repl.Console, pipeline, format, os, arch string) (string, error) {
		artifact, err := SearchArtifact(con, pipeline, "beacon", "shellcode", os, arch)
		if err != nil {
			return "", err
		}
		return string(artifact.Bin), nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "get artifact stageless shellcode",
		Input: []string{
			"pipeline: pipeline id",
			"format: reserved parameter",
			"os: os, windows only",
			"arch: arch, x64/x86",
		},
		Output: []string{
			"artifact_bin",
		},
		Example: `artifact_payload("tcp_default","raw","windows","x64")`,
	})
}
