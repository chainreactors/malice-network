package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/handler"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
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
	common.BindArgCompletions(execCmd, nil,
		carapace.ActionValues().Usage("command to execute"),
		carapace.ActionValues().Usage("arguments to the command"),
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

	common.BindArgCompletions(execAssemblyCmd, nil,
		carapace.ActionFiles().Usage("path the assembly file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

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

	common.BindArgCompletions(execShellcodeCmd, nil,
		carapace.ActionFiles().Usage("path the shellcode file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execShellcodeCmd, common.ExecuteFlagSet, common.SacrificeFlagSet, func(f *pflag.FlagSet) {
		f.String("arch", "x86", "architecture of the shellcode (x86 or x64)")
	})

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

	common.BindArgCompletions(inlineShellcodeCmd, nil,
		carapace.ActionFiles().Usage("path the shellcode file"))

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

	common.BindArgCompletions(execDLLCmd, nil,
		carapace.ActionFiles().Usage("path the DLL file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execDLLCmd, common.ExecuteFlagSet, common.SacrificeFlagSet, func(f *pflag.FlagSet) {
		f.StringP("entrypoint", "e", "entrypoint", "entrypoint")
	})

	inlineDLLCmd := &cobra.Command{
		Use:   consts.ModuleAliasInlineDll,
		Short: "Executes the given inline DLL in the current process",
		Long:  help.GetHelpFor(consts.ModuleAliasInlineDll),
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			InlineDLLCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleAliasInlineDll,
		},
	}

	common.BindArgCompletions(inlineDLLCmd, nil,
		carapace.ActionFiles().Usage("path the DLL file"))

	common.BindFlag(inlineDLLCmd, func(f *pflag.FlagSet) {
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

	common.BindArgCompletions(execPECmd, nil,
		carapace.ActionFiles().Usage("path the PE file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execPECmd, common.ExecuteFlagSet, common.SacrificeFlagSet)

	inlinePECmd := &cobra.Command{
		Use:   consts.ModuleAliasInlinePE,
		Short: "Executes the given inline PE in current process",
		Long:  help.GetHelpFor(consts.ModuleAliasInlinePE),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			InlinePECmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleAliasInlinePE,
		},
	}

	common.BindArgCompletions(inlinePECmd, nil,
		carapace.ActionFiles().Usage("path the PE file"))

	execBofCmd := &cobra.Command{
		Use:   consts.ModuleExecuteBof,
		Short: "Loads and executes Bof (Windows Only)",
		Long:  help.GetHelpFor(consts.ModuleExecuteBof),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteBofCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteBof,
		},
	}

	common.BindArgCompletions(execBofCmd, nil,
		carapace.ActionFiles().Usage("path the BOF file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

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
		Annotations: map[string]string{
			"depend": consts.ModulePowershell,
		},
	}

	common.BindArgCompletions(execPowershellCmd, nil,
		carapace.ActionFiles().Usage("path the powershell script"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execPowershellCmd)

	con.RegisterInternalFunc(
		"bshell",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, cmd string) (*clientpb.Task, error) {
			return Execute(rpc, sess, cmd)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			err := handler.HandleMaleficError(ctx.Spite)
			if err != nil {
				return "", err
			}
			resp := ctx.Spite.GetExecResponse()
			return string(resp.Stdout), nil
		},
	)

	con.RegisterInternalFunc(
		"bexecute_assembly",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path, args string) (*clientpb.Task, error) {
			return ExecAssembly(rpc, sess, path, args)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"bshinject",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, ppid int, arch, path string) (*clientpb.Task, error) {
			sac, _ := builtin.NewSacrificeProcessMessage("", int64(ppid), false, "", "")
			return ExecShellcode(rpc, sess, path, sac)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"inline_shellcode",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string) (*clientpb.Task, error) {
			return InlineShellcode(rpc, sess, path)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"bdllinject",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, ppid int, path string) (*clientpb.Task, error) {
			sac, _ := builtin.NewSacrificeProcessMessage("", int64(ppid), false, "", "")
			return ExecDLL(rpc, sess, path, "", sac)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"inline_dll",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path, entryPoint string) (*clientpb.Task, error) {
			return InlineDLL(rpc, sess, path, entryPoint)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"execute_pe",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
			return ExecPE(rpc, sess, path, sac)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"inline_pe",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string) (*clientpb.Task, error) {
			return InlinePE(rpc, sess, path)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"binline_execute",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string, args string) (*clientpb.Task, error) {
			return ExecBof(rpc, sess, path, args)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"bpowershell",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string) (*clientpb.Task, error) {
			return ExecPowershell(rpc, sess, path)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})

	return []*cobra.Command{
		execCmd,
		execAssemblyCmd,
		execShellcodeCmd,
		inlineShellcodeCmd,
		execDLLCmd,
		inlineDLLCmd,
		execPECmd,
		inlinePECmd,
		execBofCmd,
		execPowershellCmd,
	}
}
