package build

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
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

// Create a new profile with external file
profile load /path/to/profile.zip --name my_profile --pipeline pipeline_name
~~~`,
	}
	common.BindFlag(loadProfileCmd, common.ProfileSet)
	//loadProfileCmd.MarkFlagRequired("pipeline")
	loadProfileCmd.MarkFlagRequired("name")
	common.BindFlagCompletions(loadProfileCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues().Usage("profilename")
		//comp["target"] = common.BuildTargetCompleter(con)
		comp["pipeline"] = common.AllPipelineCompleter(con)
		comp["rem"] = common.RemPipelineCompleter(con)
		//comp["pulse-pipeline"] = common.AllPipelineCompleter(con)
		//comp["proxy"] = carapace.ActionValues().Usage("proxy, socks5 or http")
		//comp["obfuscate"] = carapace.ActionValues("true", "false")
		//comp["modules"] = carapace.ActionValues().Usage("e.g.: execute_exe,execute_dll")

		//comp["interval"] = carapace.ActionValues("5")
		//comp["jitter"] = carapace.ActionValues("0.2")
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
// create a default profile for <tcp/http pipeline>
profile new --name tcp_profile_demo --pipeline tcp_default

// create a default profile for rem
profile new --name rem_profile_demo --pipeline tcp_default --rem rem_default
~~~
`,
	}
	common.BindFlag(newProfileCmd, common.ProfileSet)
	// newProfileCmd.MarkFlagRequired("pipeline")
	newProfileCmd.MarkFlagRequired("name")
	common.BindFlagCompletions(newProfileCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues().Usage("profile name")
		comp["pipeline"] = common.AllPipelineCompleter(con)
		comp["rem"] = common.RemPipelineCompleter(con)
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
profile delete profile_name
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

	buildCmd.PersistentFlags().Bool("auto-download", false, "auto download artifact")

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
build beacon --addresses "https://127.0.0.1:443" --target x86_64-pc-windows-gnu --source docker

// Specify a module
build beacon --addresses "https://127.0.0.1:443,https://10.0.0.1:443" --target x86_64-pc-windows-gnu --modules nano --source docker

// Build a beacon with custom rem
build beacon --addresses "tcp://127.0.0.1:5001" --rem "tcp://nonenonenonenone:@127.0.0.1:12345?wrapper=qu7tnG..." --target x86_64-pc-windows-gnu --source action

// Build a beacon with a profile
build beacon --profile tcp_default --target x86_64-pc-windows-gnu

// Build a beacon by saas
build beacon --profile tcp_default --target x86_64-pc-windows-gnu --source saas

// Build by GithubAction
build beacon --profile tcp_default --target x86_64-pc-windows-gnu --source action
~~~`,
	}
	common.BindFlag(beaconCmd,
		common.GenerateFlagSet,
		common.GithubFlagSet,
		BeaconFlagSet,
		ProxyFlagSet,
		ModuleFlagSet,
		AntiFlagSet,
		GuardrailFlagSet,
		OllvmFlagSet,
	)
	beaconCmd.MarkFlagRequired("target")
	//beaconCmd.MarkFlagRequired("profile")
	common.BindFlagCompletions(beaconCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
		comp["source"] = common.BuildResourceCompleter(con)
		comp["autorun"] = carapace.ActionFiles().Usage("autorun zip path")
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
build bind --target x86_64-pc-windows-gnu --profile tcp_default

// Build a bind payload with additional modules
build bind --target x86_64-pc-windows-gnu --profile tcp_default --modules base,sys_full

// Build a bind payload by saas 
build bind --target x86_64-pc-windows-gnu --profile tcp_default --source saas
~~~`,
	}

	common.BindFlag(bindCmd, common.GenerateFlagSet, common.GithubFlagSet, BeaconFlagSet)
	bindCmd.MarkFlagRequired("target")
	bindCmd.MarkFlagRequired("profile")
	common.BindFlagCompletions(bindCmd, func(comp carapace.ActionMap) {

		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
		comp["source"] = common.BuildResourceCompleter(con)
	})

	bindCmd.Hidden = true

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
build prelude --target x86_64-pc-windows-gnu --profile tcp_default --autorun /path/to/autorun.zip
	
// Build a prelude payload by docker
build prelude --target x86_64-pc-windows-gnu --profile tcp_default --autorun /path/to/autorun.zip --source docker
	
// Build a prelude payload by saas
build prelude --target x86_64-pc-windows-gnu --profile tcp_default --autorun /path/to/autorun.zip --source saas
~~~`,
	}

	common.BindFlag(preludeCmd, common.GenerateFlagSet, common.GithubFlagSet, PreludeFlagSet)
	preludeCmd.MarkFlagRequired("target")
	//preludeCmd.MarkFlagRequired("profile")
	preludeCmd.MarkFlagRequired("autorun")
	common.BindFlagCompletions(preludeCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
		comp["autorun"] = carapace.ActionFiles().Usage("autorun zip path")
		comp["source"] = common.BuildResourceCompleter(con)
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
build modules --target x86_64-pc-windows-gnu --profile tcp_default

// Compile a predefined feature set of modules (nano)
build modules --target x86_64-pc-windows-gnu --profile tcp_default --modules nano

// Compile specific modules into DLLs
build modules --target x86_64-pc-windows-gnu --profile tcp_default --modules base,execute_dll

// Compile third party module(curl, rem)
build modules --3rd rem --target x86_64-pc-windows-gnu --profile tcp_default

// Compile module by saas
build modules --target x86_64-pc-windows-gnu --profile tcp_default --source saas
~~~`,
	}

	common.BindFlag(modulesCmd, common.GenerateFlagSet, common.GithubFlagSet, ModuleFlagSet)

	common.BindFlagCompletions(modulesCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
		comp["source"] = common.BuildResourceCompleter(con)
	})

	modulesCmd.MarkFlagRequired("target")
	//modulesCmd.MarkFlagRequired("profile")

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
build pulse --target x86_64-pc-windows-gnu --profile tcp_default

// Build a pulse payload by specifying artifact
build pulse --target x86_64-pc-windows-gnu --profile tcp_default --artifact-id 1
~~~
`,
	}
	common.BindFlag(pulseCmd, common.GenerateFlagSet, common.GithubFlagSet, PulseFlagSet)
	pulseCmd.MarkFlagRequired("target")
	//pulseCmd.MarkFlagRequired("address")
	//pulseCmd.MarkFlagRequired("profile")
	common.BindFlagCompletions(pulseCmd, func(comp carapace.ActionMap) {
		comp["profile"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
		comp["source"] = common.BuildResourceCompleter(con)
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
build log artifact_name --limit 70
~~~
`,
	}
	common.BindFlag(logCmd, func(f *pflag.FlagSet) {
		f.Int("limit", 50, "limit of rows")
	})
	common.BindArgCompletions(logCmd, nil, common.ArtifactCompleter(con))

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
		Annotations: map[string]string{
			"resource": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListArtifactCmd(cmd, con)
		},
		Example: `~~~
// List all available build artifacts on the server
artifact list

// Navigate the artifact table and press enter to download a specific artifact
~~~`,
	}
	showArtifactCmd := &cobra.Command{
		Use:   consts.CommandArtifactShow,
		Short: "show artifact info and profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ArtifactShowCmd(cmd, con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
		Example: `~~~
artifact show artifact_name

artifact show artifact_name --profile
~~~`,
	}

	common.BindFlag(showArtifactCmd, func(f *pflag.FlagSet) {
		f.Bool("profile", false, "show profile")
	})
	common.BindArgCompletions(showArtifactCmd, nil, common.ArtifactCompleter(con))

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

// Download an artifact in a specific format (e.g.raw, bin, golang source, C source, etc.)
  	artifact download artifact_name --format raw
`,
	}
	common.BindFlag(downloadCmd, func(f *pflag.FlagSet) {
		f.StringP("output", "o", "", "output path")
		f.StringP("format", "f", "executable", "the format of the artifact")
		f.String("RDI", "", "RDI type")
	})
	common.BindArgCompletions(downloadCmd, nil, common.ArtifactCompleter(con))
	common.BindFlagCompletions(downloadCmd, func(comp carapace.ActionMap) {
		comp["format"] = common.ArtifactFormatCompleter()
	})

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
		f.StringP("type", "t", "", "Set type")
		f.StringP("name", "n", "", "alias name")
		f.StringP("target", "", "", "rust target")
		f.StringP("comment", "c", "", "comment for artifact")
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

	artifactCmd.AddCommand(listArtifactCmd, showArtifactCmd, downloadCmd, uploadCmd, deleteCommand)

	return []*cobra.Command{profileCmd, buildCmd, artifactCmd}
}

func Register(con *core.Console) {
	con.EventCallback[consts.CtrlArtifactDownload] = func(event *clientpb.Event) {
		err := WriteOriginArtifact(con, event.Job.Name)
		if err != nil {
			con.Log.Errorf("write artifact %s error: %s", event.Job.Name, err)
			return
		}
	}
	con.RegisterServerFunc("search_artifact",
		SearchArtifact,
		&mals.Helper{
			Group: intermediate.ArtifactGroup,
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

	con.RegisterServerFunc("get_artifact",
		func(con *core.Console, sess *client.Session, format string) (*clientpb.Artifact, error) {
			artifact := &clientpb.Artifact{Name: sess.Name}
			artifact, err := con.Rpc.FindArtifact(sess.Context(), artifact)
			if err != nil {
				return nil, err
			}
			return artifact, nil
		},
		&mals.Helper{
			Group: intermediate.ArtifactGroup,
			Short: "get artifact with session self",
			Input: []string{
				"sess: session",
				"format: only support shellcode",
			},
			Output: []string{
				"builder",
			},
		})

	con.RegisterServerFunc("upload_artifact",
		UploadArtifact,
		&mals.Helper{
			Group: intermediate.ArtifactGroup,
			Short: "upload local bin to server build",
		},
	)

	con.RegisterServerFunc("download_artifact",
		DownloadArtifact,
		&mals.Helper{
			Group: intermediate.ArtifactGroup,
			Short: "download artifact with special build id",
		},
	)

	con.RegisterServerFunc("delete_artifact",
		DeleteArtifact,
		&mals.Helper{
			Group: intermediate.ArtifactGroup,
			Short: "delete artifact with special build name",
		},
	)

	con.RegisterServerFunc("self_artifact",
		func(con *core.Console, sess *client.Session) (string, error) {
			artifact := &clientpb.Artifact{
				Name: sess.Name,
			}
			artifact, err := con.Rpc.FindArtifact(sess.Context(), artifact)
			if err != nil {
				return "", err
			}
			return string(artifact.Bin), nil
		},
		&mals.Helper{
			Group: intermediate.ArtifactGroup,
			Short: "get artifact with session self",
			Input: []string{
				"sess: session",
			},
			Output: []string{
				"artifact",
			},
		})

	con.RegisterServerFunc("self_stager",
		func(con *core.Console, sess *client.Session) (string, error) {
			artifact, err := SearchArtifact(con, sess.PipelineId, "pulse", "raw", sess.Os.Name, sess.Os.Arch)
			if err != nil {
				return "", err
			}
			return string(artifact.Bin), nil
		},
		&mals.Helper{
			Group: intermediate.ArtifactGroup,
			Short: "get self artifact stager shellcode",
			Input: []string{
				"sess: session",
			},
			Output: []string{
				"artifact_bin",
			},
			Example: `self_payload(active())`,
		},
	)

	con.RegisterServerFunc("artifact_stager", func(con *core.Console, pipeline, format, os, arch string) (string, error) {
		artifact, err := SearchArtifact(con, pipeline, "pulse", "shellcode", os, arch)
		if err != nil {
			return "", err
		}
		return string(artifact.Bin), nil
	}, &mals.Helper{
		Group: intermediate.ArtifactGroup,
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

	con.RegisterServerFunc("self_payload", func(con *core.Console, sess *client.Session) (string, error) {
		artifact, err := SearchArtifact(con, sess.PipelineId, "beacon", "shellcode", sess.Os.Name, sess.Os.Arch)
		if err != nil {
			return "", fmt.Errorf("get artifact error: %s", err)
		}
		return string(artifact.Bin), nil
	}, &mals.Helper{
		Group: intermediate.ArtifactGroup,
		Short: "get self artifact stageless shellcode",
		Input: []string{
			"sess: Session",
		},
		Output: []string{
			"artifact_bin",
		},
		Example: `self_payload(active())`,
	})

	con.RegisterServerFunc("artifact_payload", func(con *core.Console, pipeline, format, os, arch string) (string, error) {
		artifact, err := SearchArtifact(con, pipeline, "beacon", "shellcode", os, arch)
		if err != nil {
			return "", err
		}
		return string(artifact.Bin), nil
	}, &mals.Helper{
		Group: intermediate.ArtifactGroup,
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
