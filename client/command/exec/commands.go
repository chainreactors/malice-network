package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
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
		Use:   consts.ModuleExecution + " [cmdline]",
		Short: "Execute commands",
		Long:  `Exec implant local executable file`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecution,
		},
		Example: `Execute the executable file without any '-' arguments.
~~~
exec whoami
~~~
Execute the executable file with '-' arguments, you need add "--" before the arguments
~~~
exec gogo.exe -- -i 127.0.0.1 -p http
~~~
`,
	}
	common.BindArgCompletions(execCmd, nil,
		carapace.ActionValues().Usage("command to execute"),
		carapace.ActionValues().Usage("arguments to the command"),
	)

	common.BindFlag(execCmd, func(f *pflag.FlagSet) {
		f.BoolP("quiet", "q", false, "disable output")
	})
	execLocalCmd := &cobra.Command{
		Use:   consts.ModuleExecuteLocal + " [local_exe]",
		Short: "Execute local PE on sacrifice process",
		Long: `
Execute local PE on sacrifice process, support spoofing process arguments, spoofing ppid, block-dll, disable etw
		`,
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteLocalCmd(cmd, con)
		},
		Args: cobra.MinimumNArgs(1),
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteLocal,
			"os":     "windows",
		},
		Example: `
~~~
execute_local local_exe --ppid 1234 --block_dll --etw --argue "argue"
~~~
`,
	}
	common.BindFlag(execLocalCmd, common.SacrificeFlagSet, func(f *pflag.FlagSet) {
		f.StringP("process", "n", "", "custom process path")
		f.BoolP("quit", "q", false, "disable output")
	})

	shellCmd := &cobra.Command{
		Use:   consts.ModuleAliasShell + " [cmdline]",
		Short: "Execute cmd",
		Long:  `equal: exec cmd /c "[cmdline]"`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ShellCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecution,
			"os":     "windows",
		},
	}

	common.BindArgCompletions(shellCmd, nil,
		carapace.ActionValues().Usage("cmd to execute"),
		carapace.ActionValues().Usage("arguments to the command"),
	)

	common.BindFlag(shellCmd, func(f *pflag.FlagSet) {
		f.BoolP("quiet", "q", false, "disable output")
	})

	powershellCmd := &cobra.Command{
		Use:   consts.ModuleAliasPowershell + " [cmdline]",
		Short: "Execute cmd with powershell",
		Long:  `equal: powershell.exe -ExecutionPolicy Bypass -w hidden -nop "[cmdline]"`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			PowershellCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecution,
			"os":     "windows",
		},
	}
	common.BindArgCompletions(powershellCmd, nil,
		carapace.ActionValues().Usage("powershell to execute"),
		carapace.ActionValues().Usage("arguments to the command"),
	)

	common.BindFlag(powershellCmd, func(f *pflag.FlagSet) {
		f.BoolP("quiet", "q", false, "disable output")
	})

	execAssemblyCmd := &cobra.Command{
		Use:   consts.ModuleExecuteAssembly + " [file]",
		Short: "Loads and executes a .NET assembly in implant process (Windows Only)",
		Long: `
Load CLR assembly in implant process(will not create new process)

if return 0x80004005, please use --amsi bypass.
`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteAssemblyCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteAssembly,
		},
		Example: `Execute a .NET assembly without "-" arguments
~~~
execute-assembly --amsi potato.exe "whoami" 
~~~
Execute a .NET assembly with "-" arguments, you need add "--" before the arguments
~~~
execute-assembly --amsi potato.exe -- -cmd "cmd /c whoami"
~~~
`,
	}

	common.BindArgCompletions(execAssemblyCmd, nil,
		carapace.ActionFiles().Usage("path the assembly file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execAssemblyCmd, common.CLRFlagSet)

	execShellcodeCmd := &cobra.Command{
		Use:   consts.ModuleExecuteShellcode + " [shellcode_file]",
		Short: "Executes the given shellcode in the sacrifice process",
		Long: `The current shellcode injection method uses APC.

In the future, configurable shellcode injection settings will be provided, along with Donut, SGN, SRDI, etc.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteShellcodeCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteShellcode,
		},
		Example: `
~~~
execute_shellcode example.bin
~~~
`,
	}

	common.BindArgCompletions(execShellcodeCmd, nil,
		carapace.ActionFiles().Usage("path the shellcode file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execShellcodeCmd, common.ExecuteFlagSet, common.SacrificeFlagSet)

	inlineShellcodeCmd := &cobra.Command{
		Use:   consts.ModuleAliasInlineShellcode + " [shellcode_file]",
		Short: "Executes the given inline shellcode in the implant process",
		Long: `
The current shellcode injection method uses APC.

!!! important ""instability warning!!!"
	inline execute shellcode may cause the implant to crash, please use with caution.
`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			InlineShellcodeCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteShellcode,
		},
		Example: `
~~~
inline_shellcode example.bin
~~~
`,
	}

	common.BindArgCompletions(inlineShellcodeCmd, nil,
		carapace.ActionFiles().Usage("path the shellcode file"))
	common.BindFlag(inlineShellcodeCmd, common.ExecuteFlagSet)

	execDLLCmd := &cobra.Command{
		Use:   consts.ModuleExecuteDll + " [dll]",
		Short: "Executes the given DLL in the sacrifice process",
		Long: `
use a custom Headless PE loader to load DLL in the sacrificed process.
`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteDLLCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteDll,
		},
		Example: `
~~~
execute_dll example.dll 
~~~

if entrypoint not default, you can specify the entrypoint

~~~
execute_dll example.dll -e entrypoint -- arg1 arg2
~~~
`,
	}

	common.BindArgCompletions(execDLLCmd, nil,
		carapace.ActionFiles().Usage("path the DLL file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execDLLCmd, common.ExecuteFlagSet, common.SacrificeFlagSet, func(f *pflag.FlagSet) {
		f.StringP("entrypoint", "e", "", "custom entrypoint")
	})

	inlineDLLCmd := &cobra.Command{
		Use:   consts.ModuleAliasInlineDll + " [dll]",
		Short: "Executes the given inline DLL in the current process",
		Long: `
use a custom Headless PE loader to load DLL in the current process.

!!! important ""instability warning!!!"
	inline execute dll may cause the implant to crash, please use with caution.
`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			InlineDLLCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteDll,
		},
		Example: `execute an inline DLL with the default entry point
~~~
inline_dll example.dll
~~~
specify the entrypoint
~~~
inline_dll example.dll -e RunFunction -- arg1 arg2
~~~`,
	}

	common.BindArgCompletions(inlineDLLCmd, nil,
		carapace.ActionFiles().Usage("path the DLL file"))

	common.BindFlag(inlineDLLCmd, common.ExecuteFlagSet, func(f *pflag.FlagSet) {
		f.StringP("entrypoint", "e", "", "entrypoint")
	})

	execExeCmd := &cobra.Command{
		Use:   consts.ModuleExecuteExe + " [exe]",
		Short: "Executes the given PE in the sacrifice process",
		Long:  `use a custom Headless PE loader to load EXE in the sacrificed process.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteExeCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteExe,
		},
		Example: `
~~~
execute_exe gogo.exe -- -i 123.123.123.123 -p top2
~~~
`,
	}

	common.BindArgCompletions(execExeCmd, nil,
		carapace.ActionFiles().Usage("path the PE file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	common.BindFlag(execExeCmd, common.ExecuteFlagSet, common.SacrificeFlagSet)

	inlinePECmd := &cobra.Command{
		Use:   consts.ModuleAliasInlineExe + " [exe]",
		Short: "Executes the given inline EXE in current process",
		Long: `
use a custom Headless PE loader to load EXE in the current process.

!!! important ""instability warning!!!"
	inline execute exe may cause the implant to crash, please use with caution.
	
	if double run same exe, More likely to crash
`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			InlineExeCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteExe,
		},
		Example: `execute the inline PE file
~~~
inline_exe hackbrowserdata.exe -- -h
~~~
`,
	}
	common.BindFlag(inlinePECmd, common.ExecuteFlagSet)
	common.BindArgCompletions(inlinePECmd, nil,
		carapace.ActionFiles().Usage("path the PE file"))

	execBofCmd := &cobra.Command{
		Use:   consts.ModuleExecuteBof + " [bof]",
		Short: "COFF Loader,  executes Bof (Windows Only)",
		Long: `
refactor from https://github.com/hakaioffsec/coffee ,fix a bundle bugs

Arguments for the BOF can be passed after the -- delimiter. Each argument must be prefixed with the type of the argument followed by a colon (:). The following types are supported:

* str - A null-terminated string
* wstr - A wide null-terminated string
* int - A signed 32-bit integer
* short - A signed 16-bit integer
* bin - A base64-encoded binary blob
`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteBofCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleExecuteBof,
		},
		Example: `
~~~
bof dir.x64.o -- wstr:"C:\\Windows\\System32"
~~~`,
	}

	common.BindArgCompletions(execBofCmd, nil,
		carapace.ActionFiles().Usage("path the BOF file"),
		carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint"))

	powerpickCmd := &cobra.Command{
		Use:   consts.ModulePowerpick + " [args]",
		Short: "unmanaged powershell on implant process (Windows Only)",
		Run: func(cmd *cobra.Command, args []string) {
			ExecutePowershellCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModulePowerpick,
		},
		Example: `
~~~
powerpick -s powerview.ps1 -- Get-NetUser
~~~
`,
	}

	common.BindFlag(powerpickCmd, common.CLRFlagSet, func(f *pflag.FlagSet) {
		f.StringP("script", "s", "", "powershell script")
	})

	common.BindArgCompletions(powerpickCmd, nil,
		carapace.ActionValues().Usage("powershell script path"))

	common.BindFlagCompletions(powerpickCmd, func(comp carapace.ActionMap) {
		comp["script"] = carapace.ActionFiles()
	})

	return []*cobra.Command{
		execCmd,
		execLocalCmd,
		shellCmd,
		powershellCmd,
		execAssemblyCmd,
		execShellcodeCmd,
		inlineShellcodeCmd,
		execDLLCmd,
		inlineDLLCmd,
		execExeCmd,
		inlinePECmd,
		execBofCmd,
		powerpickCmd,
	}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleAliasShell,
		Shell,
		"bshell",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string) (*clientpb.Task, error) {
			return Shell(rpc, sess, cmd, true)
		},
		common.ParseExecResponse,
		nil,
	)

	con.RegisterImplantFunc(
		consts.ModuleExecution,
		Execute,
		"",
		nil,
		common.ParseExecResponse,
		nil,
	)

	con.RegisterImplantFunc(
		consts.ModuleAliasPowershell,
		Powershell,
		"bpowershell",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmdline string) (*clientpb.Task, error) {
			return Powershell(rpc, sess, cmdline, true)
		},
		common.ParseExecResponse,
		nil,
	)

	con.RegisterImplantFunc(
		consts.ModuleExecuteAssembly,
		ExecAssembly,
		"bexecute_assembly",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path, args string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecAssembly(rpc, sess, path, cmdline, true, true, true)
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleExecuteShellcode,
		ExecShellcode,
		"bshinject",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, ppid int, arch, path string) (*clientpb.Task, error) {
			sac, _ := builtin.NewSacrificeProcessMessage(int64(ppid), false, true, true, "")
			return ExecShellcode(rpc, sess, path, nil, true, math.MaxUint32, sess.Os.Arch, "", sac)
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleAliasInlineShellcode,
		InlineShellcode,
		"binline_shellcode",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string) (*clientpb.Task, error) {
			return InlineShellcode(rpc, sess, path, nil, true, math.MaxUint32, sess.Os.Arch, "")
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleExecuteDll,
		ExecDLL,
		"bdllinject",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, ppid int, path string) (*clientpb.Task, error) {
			sac, _ := builtin.NewSacrificeProcessMessage(int64(ppid), false, true, true, "")
			return ExecDLL(rpc, sess, path, "DLLMain", nil, true, math.MaxUint32, sess.Os.Arch, "", sac)
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleAliasInlineDll,
		InlineDLL,
		"binline_dll",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path, entryPoint string, args string) (*clientpb.Task, error) {
			param, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return InlineDLL(rpc, sess, path, entryPoint, param, true, math.MaxUint32, sess.Os.Arch, "")
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleExecuteExe,
		ExecExe,
		"bexecute_exe",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecExe(rpc, sess, path, cmdline, true, math.MaxUint32, sess.Os.Arch, "", sac)
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleAliasInlineExe,
		InlineExe,
		"binline_exe",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args string) (*clientpb.Task, error) {
			param, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return InlineExe(rpc, sess, path, param, true, math.MaxUint32, sess.Os.Arch, "")
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleExecuteBof,
		ExecBof,
		"binline_execute",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecBof(rpc, sess, path, cmdline, true)
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModulePowerpick,
		PowerPick,
		"bpowerpick",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, script string, ps string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(ps)
			if err != nil {
				return nil, err
			}
			return PowerPick(rpc, sess, script, cmdline, true, true)
		},
		common.ParseAssembly,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleExecuteLocal,
		ExecLocal,
		"bexecute",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmdline string) (*clientpb.Task, error) {
			args, err := shellquote.Split(cmdline)
			if err != nil {
				return nil, err
			}
			return ExecLocal(rpc, sess, args, true, "", &implantpb.SacrificeProcess{
				Hidden:   false,
				BlockDll: false,
				Etw:      false,
				Ppid:     0,
				Argue:    "",
			})
		},
		common.ParseExecResponse,
		nil,
	)
}
