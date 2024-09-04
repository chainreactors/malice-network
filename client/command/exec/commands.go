package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
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
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecution,
		},
	}
	carapace.Gen(execCmd).PositionalCompletion(
		carapace.ActionValues().Usage("command to execute"),
		carapace.ActionValues().Usage("arguments to the command eg: 'arg1 arg2 arg3'"),
	)

	common.BindFlag(execCmd, common.ExecuteFlagSet)

	execAssemblyCmd := &cobra.Command{
		Use:   consts.ModuleExecuteAssembly,
		Short: "Loads and executes a .NET assembly in a child process (Windows Only)",
		Long:  help.GetHelpFor(consts.ModuleExecuteAssembly),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteAssemblyCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteAssembly,
		},
	}
	carapace.Gen(execAssemblyCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the assembly file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint, eg: 'arg1,arg2,arg3'"),
	)

	common.BindFlag(execAssemblyCmd, common.ExecuteFlagSet)

	execShellcodeCmd := &cobra.Command{
		Use:   consts.ModuleExecuteShellcode,
		Short: "Executes the given shellcode in the malefic process",
		Long:  help.GetHelpFor(consts.ModuleExecuteShellcode),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteShellcodeCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteShellcode,
		},
	}

	carapace.Gen(execShellcodeCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the shellcode file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)

	common.BindFlag(execShellcodeCmd, common.ExecuteFlagSet, common.SacrificeFlagSet)

	inlineShellcodeCmd := &cobra.Command{
		Use:   consts.ModuleAliasInlineShellcode,
		Short: "Executes the given inline shellcode in the IOM ",
		Long:  help.GetHelpFor(consts.ModuleExecuteShellcode),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			InlineShellcodeCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleAliasInlineShellcode,
		},
	}

	carapace.Gen(inlineShellcodeCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the shellcode file"),
	)

	execDLLCmd := &cobra.Command{
		Use:   consts.ModuleExecuteDll,
		Short: "Executes the given DLL in the sacrifice process",
		Long:  help.GetHelpFor(consts.ModuleExecuteDll),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteDLLCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteDll,
		},
	}

	carapace.Gen(execDLLCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the DLL file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)

	common.BindFlag(execDLLCmd, common.ExecuteFlagSet, common.SacrificeFlagSet, func(f *pflag.FlagSet) {
		f.StringP("entrypoint", "e", "entrypoint", "entrypoint")
	})

	execPECmd := &cobra.Command{
		Use:   consts.ModuleExecutePE,
		Short: "Executes the given PE in the sacrifice process",
		Long:  help.GetHelpFor(consts.ModuleExecutePE),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecutePECmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecutePE,
		},
	}

	carapace.Gen(execPECmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the PE file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)
	common.BindFlag(execPECmd, common.ExecuteFlagSet, common.SacrificeFlagSet)

	execBofCmd := &cobra.Command{
		Use:   consts.ModuleExecuteBof,
		Short: "Loads and executes Bof (Windows Only)",
		Long:  help.GetHelpFor(consts.ModuleExecuteBof),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteBofCmd(cmd, con)
			return
		},
	}

	carapace.Gen(execBofCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the BOF file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)
	common.BindFlag(execBofCmd)

	execPowershellCmd := &cobra.Command{
		Use:   consts.ModulePowershell,
		Short: "Loads and executes powershell (Windows Only)",
		Long:  help.GetHelpFor(consts.ModulePowershell),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecutePowershellCmd(cmd, con)
			return
		},
	}

	carapace.Gen(execPowershellCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("path the powershell script"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"),
	)
	common.BindFlag(execPowershellCmd)

	//&grumble.Command{
	//	Name: consts.ModuleAliasInlineDll,
	//	Help: "Executes the given inline DLL in current process",
	//	Args: func(a *grumble.Args) {
	//		a.String("path", "path the shellcode file")
	//		a.StringList("args", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
	//	},
	//	Run: func(c *grumble.Context) error {
	//		InlineDLLCmd(c, con)
	//		return nil
	//	},
	//	Help
	//	Group: consts.ImplantGroup,
	//	Completer: func(prefix string, args []string) []string {
	//		if len(args) < 2 {
	//			return completer.LocalPathCompleter(prefix, args, con)
	//		}
	//		return nil
	//	},
	//},

	//&grumble.Command{
	//	Name: consts.ModuleAliasInlinePE,
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
