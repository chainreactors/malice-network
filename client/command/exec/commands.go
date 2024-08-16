package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name:     consts.ModuleExecution,
			Help:     "Execute command",
			LongHelp: help.GetHelpFor("exec"),
			Flags: func(f *grumble.Flags) {
				f.Bool("o", "output", true, "capture command output")
				f.Int("t", "timeout", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")
			},
			Args: func(a *grumble.Args) {
				a.String("command", "command to execute")
				a.StringList("arguments", "arguments to the command")
			},
			Run: func(ctx *grumble.Context) error {
				ExecuteCmd(ctx, con)
				return nil
			},
		},

		&grumble.Command{
			Name:     consts.ModuleExecuteAssembly,
			Help:     "Loads and executes a .NET assembly in a child process (Windows Only)",
			LongHelp: help.GetHelpFor(consts.ModuleExecuteAssembly),
			Args: func(a *grumble.Args) {
				a.String("path", "path the assembly file")
				a.StringList("args", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
			},
			Flags: func(f *grumble.Flags) {
				//f.String("p", "process", "notepad.exe", "hosting process to inject into")
				//f.String("m", "method", "", "Optional method (a method is required for a .NET DLL)")
				//f.String("c", "class", "", "Optional class name (required for .NET DLL)")
				//f.String("d", "app-domain", "", "AppDomain name to create for .NET assembly. Generated randomly if not set.")
				//f.String("a", "arch", "x84", "Assembly target architecture: x86, x64, x84 (x86+x64)")
				//f.Bool("i", "in-process", false, "Run in the current sliver process")
				//f.String("r", "runtime", "", "Runtime to use for running the assembly (only supported when used with --in-process)")
				//f.Bool("s", "save", false, "save output to file")
				f.Bool("o", "output", false, "need output")
				f.String("n", "process", "C:\\Windows\\System32\\notepad.exe", "custom process path")
				f.Uint("p", "ppid", 0, "parent process id (optional)")
				//f.Bool("M", "amsi-bypass", false, "Bypass AMSI on Windows (only supported when used with --in-process)")
				//f.Bool("E", "etw-bypass", false, "Bypass ETW on Windows (only supported when used with --in-process)")

				//f.Int("t", "timeout", consts.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				ExecuteAssemblyCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
			Completer: func(prefix string, args []string) []string {
				if len(args) < 2 {
					return completer.LocalPathCompleter(prefix, args, con)
				}
				return nil
			},
		},

		&grumble.Command{
			Name:     consts.ModuleExecuteShellcode,
			Help:     "Executes the given shellcode in the sliver process",
			LongHelp: help.GetHelpFor(consts.ModuleExecuteShellcode),
			Run: func(ctx *grumble.Context) error {
				ExecuteShellcodeCmd(ctx, con)
				return nil
			},
			Args: func(a *grumble.Args) {
				a.String("path", "path the shellcode file")
				a.StringList("args", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))

			},
			Flags: func(f *grumble.Flags) {
				f.Uint("p", "ppid", 0, "pid of the process to inject into (0 means injection into ourselves)")
				f.Bool("b", "block_dll", false, "block dll injection")
				f.String("n", "process", "C:\\Windows\\System32\\notepad.exe", "custom process path")
				f.Bool("s", "sacrifice", false, "is need sacrifice process")
				f.String("a", "argue", "", "argue")
			},
			HelpGroup: consts.ImplantGroup,
			Completer: func(prefix string, args []string) []string {
				if len(args) < 2 {
					return completer.LocalPathCompleter(prefix, args, con)
				}
				return nil
			},
		},
		&grumble.Command{
			Name:     consts.ModuleInlineShellcode,
			Help:     "Executes the given inline shellcode in the IOM ",
			LongHelp: help.GetHelpFor(consts.ModuleInlineShellcode),
			Args: func(a *grumble.Args) {
				a.String("path", "path the shellcode file")
				a.StringList("args", "arguments to pass to the assembly entrypoint")
			},
			Run: func(ctx *grumble.Context) error {
				InlineShellcodeCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
			Completer: func(prefix string, args []string) []string {
				if len(args) < 2 {
					return completer.LocalPathCompleter(prefix, args, con)
				}
				return nil
			},
		},
		&grumble.Command{
			Name:     consts.ModuleExecuteDll,
			Help:     "Executes the given DLL in the sacrifice process",
			LongHelp: help.GetHelpFor(consts.ModuleExecuteDll),
			Args: func(a *grumble.Args) {
				a.String("path", "path the shellcode file")
				a.StringList("args", "arguments to pass to the assembly entrypoint", grumble.Default([]string{"C:\\Windows\\System32\\cmd.exe\x00"}))
			},
			Flags: func(f *grumble.Flags) {
				f.Uint("p", "ppid", 0, "pid of the process to inject into (0 means injection into ourselves)")
				f.Bool("b", "block_dll", false, "block dll injection")
				f.String("n", "process", "C:\\Windows\\System32\\notepad.exe", "custom process path")
				f.Bool("s", "sacrifice", false, "is need sacrifice process")
				f.String("e", "entrypoint", "entrypoint", "entrypoint")
				f.String("a", "argue", "", "argue")
			},
			Run: func(c *grumble.Context) error {
				ExecuteDLLCmd(c, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
			Completer: func(prefix string, args []string) []string {
				if len(args) < 2 {
					return completer.LocalPathCompleter(prefix, args, con)
				}
				return nil
			},
		},
		//&grumble.Command{
		//	Name: consts.ModuleInlineDll,
		//	Help: "Executes the given inline DLL in current process",
		//	Args: func(a *grumble.Args) {
		//		a.String("path", "path the shellcode file")
		//		a.StringList("args", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
		//	},
		//	Run: func(c *grumble.Context) error {
		//		InlineDLLCmd(c, con)
		//		return nil
		//	},
		//	HelpGroup: consts.ImplantGroup,
		//	Completer: func(prefix string, args []string) []string {
		//		if len(args) < 2 {
		//			return completer.LocalPathCompleter(prefix, args, con)
		//		}
		//		return nil
		//	},
		//},
		&grumble.Command{
			Name:     consts.ModuleExecutePE,
			Help:     "Executes the given PE in the sacrifice process",
			LongHelp: help.GetHelpFor(consts.ModuleExecutePE),
			Args: func(a *grumble.Args) {
				a.String("path", "path the shellcode file")
				a.StringList("args", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
			},
			Flags: func(f *grumble.Flags) {
				f.Uint("p", "ppid", 0, "pid of the process to inject into (0 means injection into ourselves)")
				f.Bool("b", "block_dll", false, "block dll injection")
				f.String("n", "process", "C:\\Windows\\System32\\notepad.exe", "custom process path")
				f.Bool("s", "sacrifice", false, "is need sacrifice process")
				f.String("a", "argue", "", "argue")
			},
			Run: func(c *grumble.Context) error {
				ExecutePECmd(c, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
			Completer: func(prefix string, args []string) []string {
				if len(args) < 2 {
					return completer.LocalPathCompleter(prefix, args, con)
				}
				return nil
			},
		},
		//&grumble.Command{
		//	Name: consts.ModuleInlinePE,
		//	Help: "Executes the given inline PE in current process",
		//	Args: func(a *grumble.Args) {
		//		a.String("path", "path the shellcode file")
		//		a.StringList("args", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
		//	},
		//	Run: func(c *grumble.Context) error {
		//		InlinePECmd(c, con)
		//		return nil
		//	},
		//	HelpGroup: consts.ImplantGroup,
		//	Completer: func(prefix string, args []string) []string {
		//		if len(args) < 2 {
		//			return completer.LocalPathCompleter(prefix, args, con)
		//		}
		//		return nil
		//	},
		//},
		&grumble.Command{
			Name:     consts.ModuleExecuteBof,
			Help:     "Loads and executes Bof (Windows Only)",
			LongHelp: help.GetHelpFor(consts.ModuleExecuteBof),
			Args: func(a *grumble.Args) {
				a.String("path", "path the assembly file")
				a.StringList("args", "arguments to pass to the assembly entrypoint")
			},
			Flags: func(f *grumble.Flags) {
				f.Int("t", "timeout", consts.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				ExecuteBofCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
			Completer: func(prefix string, args []string) []string {
				if len(args) < 2 {
					return completer.LocalPathCompleter(prefix, args, con)
				}
				return nil
			},
		},
		&grumble.Command{
			Name:     consts.ModulePowershell,
			Help:     "Loads and executes powershell (Windows Only)",
			LongHelp: help.GetHelpFor(consts.ModulePowershell),
			Args: func(a *grumble.Args) {
				a.StringList("args", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
			},
			Flags: func(f *grumble.Flags) {
				//f.Bool("s", "save", false, "save output to file")
				f.String("p", "path", "", "path to the powershell script")
				//f.String("A", "process-arguments", "", "arguments to pass to the hosting process")
				f.Int("t", "timeout", consts.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				ExecutePowershellCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
	}
}
