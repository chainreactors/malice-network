package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: consts.ModuleExecution,
			Help: "Execute command",
			Flags: func(f *grumble.Flags) {
				f.Bool("T", "token", false, "execute command with current token (windows only)")
				f.Bool("o", "output", false, "capture command output")
				f.Bool("s", "save", false, "save output to a file")
				f.Bool("X", "loot", false, "save output as loot")
				f.Bool("S", "ignore-stderr", false, "don't print STDERR output")
				f.String("O", "stdout", "", "remote path to redirect STDOUT to")
				f.String("E", "stderr", "", "remote path to redirect STDERR to")
				f.String("n", "name", "", "name to assign loot (optional)")
				f.Uint("P", "ppid", 0, "parent process id (optional, Windows only)")

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
			Name: consts.ModuleExecuteAssembly,
			Help: "Loads and executes a .NET assembly in a child process (Windows Only)",
			//LongHelp: help.GetHelpFor([]string{consts.ModuleExecuteAssembly}),
			Args: func(a *grumble.Args) {
				a.String("path", "path the assembly file")
				a.StringList("arguments", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
			},
			Flags: func(f *grumble.Flags) {
				f.String("p", "process", "notepad.exe", "hosting process to inject into")
				f.String("m", "method", "", "Optional method (a method is required for a .NET DLL)")
				f.String("c", "class", "", "Optional class name (required for .NET DLL)")
				f.String("d", "app-domain", "", "AppDomain name to create for .NET assembly. Generated randomly if not set.")
				f.String("a", "arch", "x84", "Assembly target architecture: x86, x64, x84 (x86+x64)")
				f.Bool("i", "in-process", false, "Run in the current sliver process")
				f.String("r", "runtime", "", "Runtime to use for running the assembly (only supported when used with --in-process)")
				f.Bool("s", "save", false, "save output to file")
				f.Bool("X", "loot", false, "save output as loot")
				f.String("n", "name", "", "name to assign loot (optional)")
				f.Uint("P", "ppid", 0, "parent process id (optional)")
				f.String("A", "process-arguments", "", "arguments to pass to the hosting process")
				f.Bool("M", "amsi-bypass", false, "Bypass AMSI on Windows (only supported when used with --in-process)")
				f.Bool("E", "etw-bypass", false, "Bypass ETW on Windows (only supported when used with --in-process)")

				f.Int("t", "timeout", consts.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				ExecuteAssemblyCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},

		&grumble.Command{
			Name: consts.ModuleExecuteShellcode,
			Help: "Executes the given shellcode in the sliver process",
			//LongHelp: help.GetHelpFor([]string{consts.ExecuteShellcodeStr}),
			Run: func(ctx *grumble.Context) error {
				ExecuteShellcodeCmd(ctx, con)
				return nil
			},
			Args: func(a *grumble.Args) {
				a.String("filepath", "path the shellcode file")
			},
			Flags: func(f *grumble.Flags) {
				//f.Bool("r", "rwx-pages", false, "Use RWX permissions for memory pages")
				f.Uint("p", "pid", 0, "Pid of process to inject into (0 means injection into ourselves)")
				f.String("n", "process", `c:\windows\system32\notepad.exe`, "Process to inject into when running in interactive mode")
				//f.Bool("i", "interactive", false, "Inject into a new process and interact with it")
				f.Bool("S", "shikata-ga-nai", false, "encode shellcode using shikata ga nai prior to execution")
				f.String("A", "architecture", "amd64", "architecture of the shellcode: 386, amd64 (used with --shikata-ga-nai flag)")
				f.Int("I", "iterations", 1, "number of encoding iterations (used with --shikata-ga-nai flag)")

				f.Int("t", "timeout", consts.DefaultTimeout, "command timeout in seconds")
			},
			HelpGroup: consts.ImplantGroup,
		},
		&grumble.Command{
			Name: consts.ModuleExecuteBof,
			Help: "Loads and executes Bof (Windows Only)",
			//LongHelp: help.GetHelpFor([]string{consts.ModuleExecuteAssembly}),
			Args: func(a *grumble.Args) {
				a.String("path", "path the assembly file")
				a.StringList("arguments", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
			},
			Flags: func(f *grumble.Flags) {
				f.Bool("s", "save", false, "save output to file")
				f.String("A", "process-arguments", "", "arguments to pass to the hosting process")
				f.Int("t", "timeout", consts.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				ExecuteAssemblyCmd(ctx, con)
				return nil
			},
			HelpGroup: consts.ImplantGroup,
		},
	}
}
