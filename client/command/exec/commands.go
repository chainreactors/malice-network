package exec

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/flags"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *console.Console) []*cobra.Command {
	execCmd := &cobra.Command{
		Use:   consts.ModuleExecution,
		Short: "Execute commands",
		Long:  help.GetHelpFor(consts.ModuleExecution),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}
	carapace.Gen(execCmd).PositionalCompletion(
		carapace.ActionValues().Usage("command to execute"),
		carapace.ActionValues().Usage("arguments to the command eg: 'arg1,arg2,arg3'"),
	)

	flags.Bind(consts.ModuleExecution, false, execCmd, func(f *pflag.FlagSet) {
		f.BoolP("output", "o", true, "capture command output")
		f.IntP("timeout", "t", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")
		f.StringP("stdout", "O", "", "stdout file")
		f.StringP("stderr", "E", "", "stderr file")
	})

	execAssemblyCmd := &cobra.Command{
		Use:   consts.ModuleExecuteAssembly,
		Short: "Loads and executes a .NET assembly in a child process (Windows Only)",
		Long:  help.GetHelpFor(consts.ModuleExecuteAssembly),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteAssemblyCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}
	carapace.Gen(execAssemblyCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the assembly file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint, eg: 'arg1,arg2,arg3'"),
	)

	flags.Bind(consts.ModuleExecuteAssembly, false, execAssemblyCmd, func(f *pflag.FlagSet) {
		f.BoolP("output", "o", false, "need output")
		//f.StringP("process", "n", "C:\\Windows\\System32\\notepad.exe", "custom process path")
		//f.UintP("ppid", "p", 0, "parent process id (optional)")
	})

	execShellcodeCmd := &cobra.Command{
		Use:   consts.ModuleExecuteShellcode,
		Short: "Executes the given shellcode in the malefic process",
		Long:  help.GetHelpFor(consts.ModuleExecuteShellcode),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteShellcodeCmd(cmd, con)
			return

		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(execShellcodeCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the shellcode file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint, eg: 'arg1,arg2,arg3'"),
	)

	flags.Bind(consts.ModuleExecuteShellcode, true, execShellcodeCmd, func(f *pflag.FlagSet) {
		f.BoolP("sacrifice", "s", false, "is need sacrifice process")
	})

	flags.Bind(consts.ModuleExecuteShellcode, false, execShellcodeCmd, func(f *pflag.FlagSet) {
		f.UintP("ppid", "p", 0, "pid of the process to inject into (0 means injection into ourselves)")
		f.BoolP("block_dll", "b", false, "block dll injection")
		f.StringP("process", "n", "C:\\Windows\\System32\\notepad.exe", "custom process path")
		f.StringP("argue", "a", "", "argue")
	})

	inlineShellcodeCmd := &cobra.Command{
		Use:   consts.ModuleInlineShellcode,
		Short: "Executes the given inline shellcode in the IOM ",
		Long:  help.GetHelpFor(consts.ModuleInlineShellcode),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			InlineShellcodeCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(inlineShellcodeCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the shellcode file"),
	)

	execDLLCmd := &cobra.Command{
		Use:   consts.ModuleExecuteDll,
		Short: "Executes the given DLL in the sacrifice process",
		Long:  help.GetHelpFor(consts.ModuleExecuteDll),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteDLLCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(execDLLCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the DLL file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)

	flags.Bind(consts.ModuleExecuteDll, true, execDLLCmd, func(f *pflag.FlagSet) {
		f.BoolP("sacrifice", "s", false, "is need sacrifice process")
	})

	flags.Bind(consts.ModuleExecuteDll, false, execDLLCmd, func(f *pflag.FlagSet) {
		f.UintP("ppid", "p", 0, "pid of the process to inject into (0 means injection into ourselves)")
		f.BoolP("block_dll", "b", false, "block dll injection")
		f.StringP("process", "n", "C:\\Windows\\System32\\notepad.exe", "custom process path")
		f.StringP("entrypoint", "e", "entrypoint", "entrypoint")
		f.StringP("argue", "a", "", "argue")
	})

	execPECmd := &cobra.Command{
		Use:   consts.ModuleExecutePE,
		Short: "Executes the given PE in the sacrifice process",
		Long:  help.GetHelpFor(consts.ModuleExecutePE),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ExecutePECmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(execPECmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the PE file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)

	flags.Bind(consts.ModuleExecutePE, true, execPECmd, func(f *pflag.FlagSet) {
		f.BoolP("sacrifice", "s", false, "is need sacrifice process")
	})

	flags.Bind(consts.ModuleExecutePE, false, execPECmd, func(f *pflag.FlagSet) {
		f.UintP("ppid", "p", 0, "pid of the process to inject into (0 means injection into ourselves)")
		f.BoolP("block_dll", "b", false, "block dll injection")
		f.StringP("process", "n", "C:\\Windows\\System32\\notepad.exe", "custom process path")
		f.StringP("argue", "a", "", "argue")
	})

	execBofCmd := &cobra.Command{
		Use:   consts.ModuleExecuteBof,
		Short: "Loads and executes Bof (Windows Only)",
		Long:  help.GetHelpFor(consts.ModuleExecuteBof),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteBofCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(execBofCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the BOF file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)

	flags.Bind(consts.ModuleExecuteBof, false, execBofCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", consts.DefaultTimeout, "command timeout in seconds")
	})

	execPowershellCmd := &cobra.Command{
		Use:   consts.ModulePowershell,
		Short: "Loads and executes powershell (Windows Only)",
		Long:  help.GetHelpFor(consts.ModulePowershell),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ExecutePowershellCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(execPowershellCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the powershell script"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)

	flags.Bind(consts.ModulePowershell, false, execPowershellCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", consts.DefaultTimeout, "command timeout in seconds")
	})

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
	return []*cobra.Command{
		execCmd,
		execAssemblyCmd,
		execShellcodeCmd,
		inlineShellcodeCmd,
		execDLLCmd,
		execPECmd,
		execBofCmd,
		execPowershellCmd,
	}
}
