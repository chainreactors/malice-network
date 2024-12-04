package exec

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/donut"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"math"
)

// ExecuteShellcodeCmd - Execute shellcode in-memory
func ExecuteShellcodeCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	task, err := ExecShellcode(con.Rpc, session, path, args, output, timeout, arch, process, common.ParseSacrificeFlags(cmd))
	if err != nil {
		return err
	}
	session.Console(task, path)
	return nil
}

func ExecShellcode(rpc clientrpc.MaliceRPCClient, sess *core.Session, shellcodePath string,
	args []string, output bool, timeout uint32, arch string, process string,
	sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	if arch == "" {
		arch = sess.Os.Arch
	}

	binary, err := common.NewBinary(consts.ModuleExecuteShellcode, shellcodePath, args, output, timeout, arch, process, sac)
	if err != nil {
		return nil, err
	}
	if pe.IsPeExt(shellcodePath) {
		cmdline := shellquote.Join(args...)
		binary.Bin, err = donut.DonutShellcodeFromPE(shellcodePath, binary.Bin, arch, cmdline, "", "", false, true)
		if err != nil {
			return nil, err
		}
		logs.Log.Infof("found pe file, auto convert to shellcode with donut")
		binary.Args = nil
	}
	task, err := rpc.ExecuteShellcode(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, err
}

func InlineShellcodeCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	task, err := InlineShellcode(con.Rpc, session, path, args, output, timeout, arch, process)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, path)
	return nil
}

func InlineShellcode(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args []string,
	output bool, timeout uint32, arch string, process string) (*clientpb.Task, error) {
	if arch == "" {
		arch = sess.Os.Arch
	}
	binary, err := common.NewBinary(consts.ModuleExecuteShellcode, path, args, output, timeout, arch, process, nil)
	if err != nil {
		return nil, err
	}
	if pe.IsPeExt(path) {
		cmdline := shellquote.Join(args...)
		binary.Bin, err = donut.DonutShellcodeFromPE(path, binary.Bin, arch, cmdline, "", "", false, true)
		if err != nil {
			return nil, err
		}
		logs.Log.Infof("found pe file, auto convert to shellcode with donut")
		//binary.Args = nil
	}
	shellcodeTask, err := rpc.ExecuteShellcode(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return shellcodeTask, err
}

func RegisterShellcodeFunc(con *repl.Console) {

	con.RegisterImplantFunc(
		consts.ModuleExecuteShellcode,
		ExecShellcode,
		"bshinject",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, ppid uint32, arch, path string) (*clientpb.Task, error) {
			return ExecShellcode(rpc, sess, path, nil, true, math.MaxUint32, sess.Os.Arch, "", common.NewSacrifice(ppid, false, true, true, ""))
		},
		common.ParseAssembly,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleExecuteShellcode,
		consts.ModuleExecuteShellcode,
		consts.ModuleExecuteShellcode+`(active(), "/path/to/shellcode", {}, true, 60, "x64", "",new_sacrifice(1234,false,true,true)`,
		[]string{
			"session: special session",
			"shellcodePath: path to shellcode",
			"args: arguments",
			"output",
			"timeout",
			"arch",
			"process",
			"sac: sacrifice process",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleAliasInlineShellcode,
		InlineShellcode,
		"binline_shellcode",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string) (*clientpb.Task, error) {
			return InlineShellcode(rpc, sess, path, nil, true, math.MaxUint32, sess.Os.Arch, "")
		},
		common.ParseAssembly,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleAliasInlineShellcode,
		consts.ModuleAliasInlineShellcode,
		consts.ModuleAliasInlineShellcode+`(active(),"/path/to/shellcode",{},true,60,"x64","")`,
		[]string{
			"session: special session",
			"path",
			"args",
			"output",
			"timeout",
			"arch",
			"process",
		},
		[]string{"task"})

}
