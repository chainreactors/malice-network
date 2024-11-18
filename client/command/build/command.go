package build

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/kballard/go-shellquote"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"strings"
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
		Use:   consts.CommandProfileNew,
		Short: "Create a new compile profile",
		Long: `Create a new compile profile with customizable attributes.

If no **name** is provided, the command concatenates the target with a random name.
- Specify the **pipeline_id** to associate the listener and pipeline.
- Specify the **target** to set the build target arch and platform.
- Use the **modules** flag to define a comma-separated list of modules, such as execute_exe or execute_dll.
- **interval** defaults to 5 seconds, controlling the execution interval of the profile.
- **jitter** adds randomness to the interval (default value is 0.2).
- The **proxy** flag allows setting up proxy configurations (e.g., http or socks5).
- **ca** enables or disables CA validation (default: disabled).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProfileNewCmd(cmd, con)
		},
		Example: `~~~
// Create a new profile with default settings
profile new --name my_profile --target x86_64-unknown-linux-musl

// Create a profile with specific modules
profile new --name my_profile --target x86_64-unknown-linux-musl --modules base,sys_full

// Create a profile with custom interval and jitter
profile new --name my_profile --target x86_64-unknown-linux-musl --interval 10 --jitter 0.5
~~~`,
	}
	common.BindFlag(newCmd, common.ProfileSet)
	common.BindFlagCompletions(newCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues("profile name")
		comp["target"] = common.BuildTargetCompleter(con)
		comp["pipeline_id"] = common.AllPipelineCompleter(con)
		comp["proxy"] = carapace.ActionValues("http", "socks5")
		//comp["obfuscate"] = carapace.ActionValues("true", "false")
		comp["modules"] = carapace.ActionValues("e.g.: execute_exe,execute_dll")
		comp["ca"] = carapace.ActionValues("true", "false")

		comp["interval"] = carapace.ActionValues("5")
		comp["jitter"] = carapace.ActionValues("0.2")
	})

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

The **target** flag is required to specify the arch and platform for the beacon, such as **x86_64-unknown-linux-musl** or **x86_64-pc-windows-msvc**.
- If **profile_name** is provided, it must match an existing compile profile. Otherwise, the command will use default settings for the beacon generation.
- Additional modules can be added to the beacon using the **modules** flag, separated by commas.
- The **shellcode_type** flag determines whether the payload should be converted into shellcode. For example, setting this flag to **srdi** triggers the conversion process.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return BeaconCmd(cmd, con)
		},
		Example: `~~~
// Build a beacon with specified settings
build beacon --target x86_64-unknown-linux-musl --profile_name beacon_profile

// Build a beacon for the Windows platform
build beacon --target x86_64-unknown-linux-musl

// Build a beacon using a specific profile and additional modules
build beacon --target x86_64-pc-windows-msvc --profile_name beacon_profile --modules base,sys_full

// Build a beacon and convert it into shellcode (srdi format)
build beacon --target x86_64-pc-windows-msvc --shellcode_type srdi
~~~`,
	}
	common.BindFlag(beaconCmd, common.GenerateFlagSet)
	common.BindFlagCompletions(beaconCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
	})
	common.BindArgCompletions(beaconCmd, nil, common.ProfileCompleter(con))

	bindCmd := &cobra.Command{
		Use:   consts.CommandBuildBind,
		Short: "Build a bind payload",
		Long: `Generate a bind payload that connects a client to the server.

The **target** flag is required to specify the target arch and platform, such as **x86_64-unknown-linux-musl** or **x86_64-pc-windows-msvc**.
- If **profile_name** is provided, it must match an existing compile profile.
- Use additional flags to include functionality such as modules or custom configurations.
- The **shellcode_type** flag can be set to specify whether the beacon should be converted into shellcode, such as **srdi**.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return BindCmd(cmd, con)
		},
		Example: `~~~
// Build a bind payload for the Windows platform
build bind --target x86_64-unknown-linux-musl

// Build a bind payload with a specific profile
build bind --target x86_64-pc-windows-msvc --profile_name bind_profile

// Build a bind payload with additional modules
build bind --target x86_64-pc-windows-msvc --modules base,sys_full

// Build a bind payload and convert it into shellcode (srdi format)
build bind --target x86_64-pc-windows-msvc --shellcode_type srdi
~~~`,
	}

	common.BindFlag(bindCmd, common.GenerateFlagSet)
	common.BindFlagCompletions(bindCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
	})
	common.BindArgCompletions(bindCmd, nil, common.ProfileCompleter(con))

	shellCodeCmd := &cobra.Command{
		Use:   consts.CommandBuildShellCode,
		Short: "build ShellCode",

		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ShellCodeCmd(cmd, con)
		},
	}

	common.BindFlag(shellCodeCmd, common.GenerateFlagSet)
	common.BindFlagCompletions(shellCodeCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
	})
	common.BindArgCompletions(shellCodeCmd, nil, common.ProfileCompleter(con))

	preludeCmd := &cobra.Command{
		Use:   consts.CommandBuildPrelude,
		Short: "Build a prelude payload",
		Long: `Generate a prelude payload as part of a multi-stage deployment.

The **target** flag is required to specify the platform, such as **x86_64-unknown-linux-musl** or **x86_64-pc-windows-msvc**, ensuring compatibility with the deployment environment.
- The **profile_name** flag is optional; if provided, it must match an existing compile profile. This allows the prelude payload to inherit settings such as interval, jitter, or proxy configurations.
- Use the **modules** flag to include additional functionalities in the payload, such as execute_exe or execute_dll. Modules should be specified as a comma-separated list.
- The **shellcode_type** flag determines whether the payload should be converted into shellcode format, such as **srdi**, enabling compatibility with specific deployment requirements.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PreludeCmd(cmd, con)
		},
		Example: `~~~
// Build a prelude payload for the Windows platform
build prelude --target x86_64-unknown-linux-musl

// Build a prelude payload with anti-sandbox and anti-debugging enabled
build prelude --target x86_64-unknown-linux-musl --modules base,sys_full

// Build a prelude payload with a specific profile
build prelude --target x86_64-pc-windows-msvc --profile_name prelude_profile

// Build a prelude payload and convert it into shellcode (srdi format)
build prelude --target x86_64-pc-windows-msvc --shellcode_type srdi
~~~`,
	}

	common.BindFlag(preludeCmd, common.GenerateFlagSet)
	common.BindFlagCompletions(preludeCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
	})
	common.BindArgCompletions(preludeCmd, nil, common.ProfileCompleter(con))

	modulesCmd := &cobra.Command{
		Use:   consts.CommandBuildModules,
		Short: "Compile specified modules into DLLs",
		Long: `Compile the specified modules into DLL files for deployment or integration.

The **target** flag is required to specify the platform for the modules, such as **x86_64-unknown-linux-musl** or **x86_64-pc-windows-msvc**,
- The **profile_name** flag is optional; if provided, it must match an existing compile profile, allowing the modules to inherit relevant configurations such as interval, jitter, or proxy settings.
- Additional modules can be explicitly defined using the **modules** flag as a comma-separated list (e.g., base, fs_mem, execute_dll). This allows fine-grained control over which modules are compiled. If **modules** is not specified, the default value will be **full**, which includes all available modules.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ModulesCmd(cmd, con)
		},
		Example: `~~~
// Compile all modules for the Windows platform
build modules --target x86_64-unknown-linux-musl

// Compile a predefined feature set of modules (nano)
build modules --target x86_64-unknown-linux-musl --modules nano

// Compile specific modules into DLLs
build modules --target x86_64-pc-windows-msvc --modules base,execute_dll

// Compile modules using a specific profile
build modules --target x86_64-pc-windows-msvc --profile_name my_profile --modules base, fs_mem
~~~`,
	}

	common.BindFlagCompletions(modulesCmd, func(comp carapace.ActionMap) {
		comp["profile_name"] = common.ProfileCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
		//comp["type"] = common.BuildFormatCompleter(con)
	})
	common.BindArgCompletions(modulesCmd, nil, common.ProfileCompleter(con))

	buildCmd.AddCommand(beaconCmd, bindCmd, preludeCmd, modulesCmd)

	srdiCmd := &cobra.Command{
		Use:   consts.CommandSRDI,
		Short: "Build SRDI artifact",
		Long: `Generate an SRDI (Shellcode Reflective DLL Injection) artifact to minimize PE (Portable Executable) signatures.

SRDI technology reduces the PE characteristics of a DLL, enabling more effective injection and evasion during execution. The following options are supported:

- The **path** flag specifies the file path to the target DLL that will be processed. This is required if **id** is not provided.
- The **id** flag identifies a specific artifact or build file in the system for conversion to SRDI format. This is required if **path** is not provided.
- The **arch** flag defines the architecture of the generated shellcode, such as **x86** or **x64**. This flag is required to ensure compatibility with the target environment.
- The **platform** flag specifies the platform of the shellcode. Defaults to **win**, but can also be set to **linux**. This flag is required to tailor the shellcode for the desired operating system.
- The **function_name** flag sets the entry function name within the DLL for execution. This is critical for specifying which function will be executed when the DLL is loaded.
- The **user_data_path** flag allows the inclusion of user-defined data to be embedded with the shellcode during generation. This can be used to pass additional information or configuration to the payload at runtime.`,
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

The following flag is supported:
- **--output**, **-o**: Specify the output path where the downloaded file will be saved. If not provided, the file will be saved in the current directory.`,
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

The following flags are supported:
- **--stage**, **-s**: Specify the stage for the artifact (eg.,: **loader**, **prelude**, **beacon**, **bind**, **modules**)
- **--type**, **-t**: Define the type of the artifact
- **--name**, **-n**: Provide an alias name for the uploaded artifact. If not provided, the server will use the original file name.`,
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
	con.RegisterServerFunc("payload_local", func(shellcodePath string) (string, error) {
		if shellcodePath != "" {
			shellcode, _ := os.ReadFile(shellcodePath)
			if _, err := os.Stat(shellcodePath); os.IsNotExist(err) {
				return "", fmt.Errorf("shellcode file does not exist: %s", shellcodePath)
			}
			return string(shellcode), nil
		} else {
			return "shellcode123", nil
		}
	}, nil)

	con.RegisterServerFunc("donut_exe2shellcode", func(exe []byte, arch string, param string) (string, error) {
		cmdline, err := shellquote.Split(param)
		if err != nil {
			return "", err
		}

		bin, err := con.Rpc.EXE2Shellcode(con.Context(), &clientpb.EXE2Shellcode{
			Bin:    exe,
			Arch:   arch,
			Type:   "donut",
			Params: strings.Join(cmdline, ","),
		})
		if err != nil {
			return "", err
		}
		return string(bin.Bin), nil
	}, nil)

	con.RegisterServerFunc("donut_dll2shellcode", func(dll []byte, arch string, param string) (string, error) {
		cmdline, err := shellquote.Split(param)
		if err != nil {
			return "", err
		}

		bin, err := con.Rpc.DLL2Shellcode(con.Context(), &clientpb.DLL2Shellcode{
			Bin:    dll,
			Arch:   arch,
			Type:   "donut",
			Params: strings.Join(cmdline, ","),
		})
		if err != nil {
			return "", err
		}
		return string(bin.Bin), nil
	}, nil)

	con.RegisterServerFunc("srdi", func(dll []byte, entry string, arch string, param string) (string, error) {
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
	}, nil)

	con.RegisterServerFunc("sgn_encode", func(shellcode []byte, arch string, iterations int32) (string, error) {
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
	}, nil)
}
