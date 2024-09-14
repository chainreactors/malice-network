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
	"github.com/kballard/go-shellquote"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"math"
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
			"depend": consts.ModuleExecuteShellcode,
		},
	}

	common.BindArgCompletions(inlineShellcodeCmd, nil,
		carapace.ActionFiles().Usage("path the shellcode file"))
	common.BindFlag(inlineShellcodeCmd, common.ExecuteFlagSet)

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
			"depend": consts.ModuleExecuteDll,
		},
	}

	common.BindArgCompletions(inlineDLLCmd, nil,
		carapace.ActionFiles().Usage("path the DLL file"))

	common.BindFlag(inlineDLLCmd, func(f *pflag.FlagSet) {
		f.StringP("entrypoint", "e", "entrypoint", "entrypoint")
	})

	execPECmd := &cobra.Command{
		Use:   consts.ModuleExecuteExe,
		Short: "Executes the given PE in the sacrifice process",
		Long:  help.GetHelpFor(consts.ModuleExecuteExe),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteExeCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteExe,
		},
	}

	common.BindArgCompletions(execPECmd, nil,
		carapace.ActionFiles().Usage("path the PE file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execPECmd, common.ExecuteFlagSet, common.SacrificeFlagSet)

	inlinePECmd := &cobra.Command{
		Use:   consts.ModuleAliasInlineExe,
		Short: "Executes the given inline EXE in current process",
		Long:  help.GetHelpFor(consts.ModuleAliasInlineExe),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			InlineExeCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteExe,
		},
	}
	common.BindFlag(inlinePECmd, common.ExecuteFlagSet)
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

	execPowershellCmd := &cobra.Command{
		Use:   consts.ModulePowershell,
		Short: "Loads and executes powershell (Windows Only)",
		Long:  help.GetHelpFor(consts.ModulePowershell),
		Run: func(cmd *cobra.Command, args []string) {
			ExecutePowershellCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModulePowershell,
		},
	}

	common.BindFlag(execPowershellCmd, func(f *pflag.FlagSet) {
		f.StringP("script", "s", "", "powershell script")
	})

	common.BindArgCompletions(execPowershellCmd, nil,
		carapace.ActionValues().Usage("powershell"))

	common.BindFlagCompletions(execPowershellCmd, func(comp carapace.ActionMap) {
		comp["script"] = carapace.ActionFiles()
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

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleExecution,
		Execute,
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

	con.RegisterImplantFunc(
		consts.ModuleExecuteAssembly,
		ExecAssembly,
		"bexecute_assembly",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path, args string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecAssembly(rpc, sess, path, cmdline, true)
		}, common.ParseAssembly)

	con.RegisterImplantFunc(
		consts.ModuleExecuteShellcode,
		ExecShellcode,
		"bshinject",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, ppid int, arch, path string) (*clientpb.Task, error) {
			sac, _ := builtin.NewSacrificeProcessMessage(int64(ppid), false, true, true, "")
			return ExecShellcode(rpc, sess, path, nil, true, math.MaxUint32, sess.Os.Arch, "", sac)
		}, common.ParseAssembly)

	con.RegisterImplantFunc(
		consts.ModuleAliasInlineShellcode,
		InlineShellcode,
		"binline_shellcode",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string) (*clientpb.Task, error) {
			return InlineShellcode(rpc, sess, path, nil, true, math.MaxUint32, sess.Os.Arch, "")
		}, common.ParseAssembly)

	con.RegisterImplantFunc(
		consts.ModuleExecuteDll,
		ExecDLL,
		"bdllinject",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, ppid int, path string) (*clientpb.Task, error) {
			sac, _ := builtin.NewSacrificeProcessMessage(int64(ppid), false, true, true, "")
			return ExecDLL(rpc, sess, path, "DLLMain", nil, true, math.MaxUint32, sess.Os.Arch, "", sac)
		}, common.ParseAssembly)

	con.RegisterImplantFunc(
		consts.ModuleAliasInlineDll,
		InlineDLL,
		"binline_dll",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path, entryPoint string, args string, process string) (*clientpb.Task, error) {
			param, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return InlineDLL(rpc, sess, path, entryPoint, param, true, math.MaxUint32, sess.Os.Arch, process)
		}, common.ParseAssembly)

	con.RegisterImplantFunc(
		consts.ModuleExecuteExe,
		ExecExe,
		"bexecute_exe",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string, args string, process string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecExe(rpc, sess, path, cmdline, true, math.MaxUint32, sess.Os.Arch, process, sac)
		}, common.ParseAssembly)

	con.RegisterImplantFunc(
		consts.ModuleAliasInlineExe,
		InlineExe,
		"binline_exe",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string, args string) (*clientpb.Task, error) {
			param, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return InlineExe(rpc, sess, path, param, true, math.MaxUint32, sess.Os.Arch, "")
		}, common.ParseAssembly)

	con.RegisterImplantFunc(
		consts.ModuleExecuteBof,
		ExecBof,
		"binline_execute",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string, args string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecBof(rpc, sess, path, cmdline, true)
		}, common.ParseAssembly)

	con.RegisterImplantFunc(
		consts.ModulePowershell,
		ExecPowershell,
		"bpowershell",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, script string, ps string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(ps)
			if err != nil {
				return nil, err
			}
			return ExecPowershell(rpc, sess, script, cmdline)
		}, common.ParseAssembly)
}
