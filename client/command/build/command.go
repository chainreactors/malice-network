package build

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/donut"
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

	newCmd := &cobra.Command{
		Use:   consts.CommandProfileLoad,
		Short: "Create a new compile profile",
		Long: `Create a new compile profile with customizable attributes.

The **profile load** command requires a valid configuration file path (e.g., **config.yaml**) to load settings. This file specifies attributes necessary for generating the compile profile.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProfileNewCmd(cmd, con)
		},
		Example: `~~~
// Create a new profile
profile load /path/to/config.yaml --name my_profile

// Create a new profile using network configuration in pipeline
profile load /path/to/config.yaml --name my_profile --pipeline pipeline_name

// Create a profile with specific modules
profile load /path/to/config.yaml --name my_profile --modules base,sys_full

// Create a profile with custom interval and jitter
profile load /path/to/config.yaml --name my_profile --interval 10 --jitter 0.5
~~~`,
	}
	common.BindFlag(newCmd, common.ProfileSet)
	newCmd.MarkFlagRequired("pipeline")
	newCmd.MarkFlagRequired("name")
	common.BindFlagCompletions(newCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues("profile name")
		//comp["target"] = common.BuildTargetCompleter(con)
		comp["pipeline"] = common.AllPipelineCompleter(con)
		//comp["proxy"] = carapace.ActionValues("").Usage("")
		//comp["obfuscate"] = carapace.ActionValues("true", "false")
		comp["modules"] = carapace.ActionValues("e.g.: execute_exe,execute_dll")
		comp["ca"] = carapace.ActionValues("true", "false")

		comp["interval"] = carapace.ActionValues("5")
		comp["jitter"] = carapace.ActionValues("0.2")
	})
	common.BindArgCompletions(newCmd, nil, carapace.ActionFiles().Usage("profile path"))

	profileCmd.AddCommand(listCmd, newCmd)

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
	
	The **target** flag is required to specify the platform, such as **x86_64-unknown-linux-musl** or **x86_64-pc-windows-msvc**, ensuring compatibility with the deployment environment.
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

	srdiCmd := &cobra.Command{
		Use:   consts.CommandSRDI,
		Short: "use srdi to generate shellcode",
		Long: `Generate an SRDI (Shellcode Reflective DLL Injection) artifact to minimize PE (Portable Executable) signatures.

SRDI technology reduces the PE characteristics of a DLL, enabling more effective injection and evasion during execution. The following options are supported:
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return SRDICmd(cmd, con)
		},
		Example: `~~~
// Convert a DLL to SRDI format with architecture and platform
srdi --path /path/to/target --arch x64 --platform win

// Specify an entry function for the DLL during SRDI conversion
srdi --path /path/to/target --arch x86 --platform linux 

// Include user-defined data with the generated shellcode
srdi --path /path/to/target.dll --arch x64 --platform win --user_data_path /path/to/user_data --function_name DllMain

// Convert a specific artifact to SRDI format using its ID
srdi --id artifact_id --arch x64 --platform linux
~~~`,
	}
	common.BindFlag(srdiCmd, common.SRDIFlagSet)
	common.BindFlagCompletions(srdiCmd, func(comp carapace.ActionMap) {
		comp["path"] = carapace.ActionFiles().Usage("file path")
		comp["id"] = common.ArtifactCompleter(con)
	})

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
	}
	common.BindFlag(downloadCmd, func(f *pflag.FlagSet) {
		f.StringP("output", "o", "", "output path")
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

	artifactCmd.AddCommand(listArtifactCmd, downloadCmd, uploadCmd)

	return []*cobra.Command{profileCmd, buildCmd, artifactCmd, srdiCmd}
}

func Register(con *repl.Console) {
	con.RegisterServerFunc("malefic_srdi", MaleficSRDI, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "malefic srdi",
	})

	con.RegisterServerFunc("find_artifact", func(con *repl.Console, profile, arch, os, typ string) (*clientpb.Builder, error) {
		return con.Rpc.FindArtifact(con.Context(), &clientpb.Builder{
			ProfileName: profile,
			Arch:        arch,
			Platform:    os,
			Type:        typ,
		})
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "find build artifact",
	})

	con.RegisterServerFunc("upload_artifact", UploadArtifact, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "upload local bin to server build",
	})

	con.RegisterServerFunc("download_artifact", DownloadArtifact, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "download artifact with special build id",
	})

	intermediate.RegisterFunction("exe2shellcode",
		func(exe []byte, arch string, cmdline string) (string, error) {
			bin, err := donut.DonutShellcodeFromPE("1.exe", exe, arch, cmdline, false, true)
			if err != nil {
				return "", err
			}
			return string(bin), nil
		})
	intermediate.AddHelper("exe2shellcode", &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "exe to shellcode with donut",
		Input: []string{
			"bin: dll bin",
			"arch: architecture",
			"param: cmd args",
		},
		Output: []string{
			"shellcode: shellcode bin",
		},
	})

	intermediate.RegisterFunction("dll2shellcode", func(dll []byte, arch string, cmdline string) (string, error) {
		bin, err := donut.DonutShellcodeFromPE("1.dll", dll, arch, cmdline, false, true)
		if err != nil {
			return "", err
		}
		return string(bin), nil
	})
	intermediate.AddHelper("dll2shellcode", &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "dll to shellcode with donut",
		Input: []string{
			"bin: dll bin",
			"arch: architecture, x86/x64",
			"param: cmd args",
		},
		Output: []string{
			"shellcode: shellcode bin",
		},
	})

	intermediate.RegisterFunction("clr2shellcode", donut.DonutFromAssemblyFromFile)
	intermediate.AddHelper("clr2shellcode", &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "clr to shellcode with donut",
		Input: []string{
			"file: path to PE file",
			"arch: architecture, x86/x64",
			"cmdline: cmd args",
			"method: name of method or DLL function to invoke for .NET DLL and unmanaged DLL",
			"classname: name of class with optional namespace for .NET DLL",
			"appdomain: name of domain to create for .NET DLL/EXE",
		},
		Output: []string{
			"shellcode: bin",
		},
	})

	intermediate.RegisterFunction("donut", donut.DonutShellcodeFromFile)
	intermediate.AddHelper("donut", &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "Generates x86, x64, or AMD64+x86 position-independent shellcode that loads .NET Assemblies, PE files, and other Windows payloads from memory and runs them with parameters ",
		Input: []string{
			"file: path to PE file",
			"arch: architecture, x86/x64",
			"cmdline: cmd args",
		},
		Output: []string{
			"shellcode",
		},
	})

	con.RegisterServerFunc("srdi", func(con *repl.Console, dll []byte, entry string, arch string, param string) (string, error) {
		bin, err := con.Rpc.DLL2Shellcode(con.Context(), &clientpb.DLL2Shellcode{
			Bin:        dll,
			Arch:       arch,
			Type:       "srdi",
			Entrypoint: entry,
			Params:     param,
		})
		if err != nil {
			return "", err
		}
		return string(bin.Bin), nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "dll/exe to shellcode with srdi",
		Input: []string{
			"bin: dll/exe bin",
			"entry: entry function for dll",
			"arch: architecture, x86/x64",
			"param: cmd args",
		},
		Output: []string{
			"shellcode: shellcode bin",
		},
	})

	con.RegisterServerFunc("sgn_encode", func(con *repl.Console, shellcode []byte, arch string, iterations int32) (string, error) {
		bin, err := con.Rpc.ShellcodeEncode(con.Context(), &clientpb.ShellcodeEncode{
			Shellcode:  shellcode,
			Arch:       arch,
			Type:       "sgn",
			Iterations: iterations,
		})
		if err != nil {
			return "", err
		}
		return string(bin.Bin), nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "shellcode encode with sgn",
		Input: []string{
			"bin: shellcode bin",
			"arch: architecture, x86/x64",
			"iterations: sgn iterations",
		},
		Output: []string{
			"shellcode: encoded shellcode bin",
		},
	})

	con.RegisterServerFunc("artifact_stager", func(con *repl.Console, profile string, arch string, os string) (string, error) {
		builder, err := con.Rpc.FindArtifact(con.Context(), &clientpb.Builder{
			ProfileName: profile,
			Arch:        arch,
			Platform:    os,
			Type:        "pulse",
		})
		if err != nil {
			return "", err
		}
		return string(builder.Bin), nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "get artifact stager shellcode",
		Input: []string{
			"profile: profile name",
			"arch: architecture, x86/x64",
			"os: platform, win/linux",
		},
		Output: []string{
			"shellcode: shellcode bin",
		},
	})

	con.RegisterServerFunc("artifact_payload", func(con *repl.Console, profile string, arch string, os string) (string, error) {
		builder, err := con.Rpc.FindArtifact(con.Context(), &clientpb.Builder{
			ProfileName: profile,
			Arch:        arch,
			Platform:    os,
			Type:        "beacon",
		})
		if err != nil {
			return "", err
		}
		return string(builder.Bin), nil
	}, &intermediate.Helper{
		Group: intermediate.GroupArtifact,
		Short: "get artifact stageless shellcode",
		Input: []string{
			"profile: profile name",
			"arch: architecture, x86/x64",
			"os: platform, win/linux",
		},
		Output: []string{
			"shellcode: shellcode bin",
		},
	})
}
