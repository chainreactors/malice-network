package exec

import (
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"math"
)

// ExecuteExeCmd - Execute PE on sacrifice process
func ExecuteExeCmd(cmd *cobra.Command, con *repl.Console) error {
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	sac, _ := common.ParseSacrificeFlags(cmd)
	task, err := ExecExe(con.Rpc, con.GetInteractive(), path, args, output, timeout, arch, process, sac)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, path)
	return nil
}

func ExecExe(rpc clientrpc.MaliceRPCClient, sess *core.Session, pePath string,
	args []string, output bool, timeout uint32, arch string,
	process string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	if arch == "" {
		arch = sess.Os.Arch
	}
	binary, err := common.NewBinary(consts.ModuleExecuteExe, pePath, args, output, timeout, arch, process, sac)
	if err != nil {
		return nil, err
	}
	if pe.CheckPEType(binary.Bin) != consts.EXEFile {
		return nil, errors.New("the file is not a EXE file")
	}
	task, err := rpc.ExecuteEXE(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// InlineExeCmd - Execute PE in current process
func InlineExeCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	task, err := InlineExe(con.Rpc, session, path, args, output, timeout, arch, process)
	if err != nil {
		return err
	}
	session.Console(task, path)
	return nil
}

func InlineExe(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args []string,
	output bool, timeout uint32, arch string, process string) (*clientpb.Task, error) {
	if arch == "" {
		arch = sess.Os.Arch
	}
	binary, err := common.NewBinary(consts.ModuleExecuteExe, path, args, output, timeout, arch, process, nil)
	if err != nil {
		return nil, err
	}
	if pe.CheckPEType(binary.Bin) != consts.EXEFile {
		return nil, errors.New("the file is not a PE file")
	}
	task, err := rpc.ExecuteEXE(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterExeFunc(con *repl.Console) {
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

	con.AddInternalFuncHelper(
		consts.ModuleAliasInlineExe,
		consts.ModuleAliasInlineExe,
		consts.ModuleAliasInlineExe+`(active(),"gogo.exe",{"-i","127.0.0.1"},true,60,"",""))`,
		[]string{
			"session: special session",
			"path: PE file",
			"args: PE args",
			"output",
			"timeout",
			"arch",
			"process",
		},
		[]string{"task"})

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

	con.AddInternalFuncHelper(
		consts.ModuleExecuteExe,
		consts.ModuleExecuteExe,
		consts.ModuleExecuteExe+`(active(),"/path/to/gogo.exe",{"-i","127.0.0.1"},true,60,"","",new_sacrifice(1234,false,true,true,"argue"))`,
		[]string{
			"session: special session",
			"pePath: PE file",
			"args: PE args",
			"output",
			"timeout",
			"arch",
			"process",
			"sac: sacrifice process",
		},
		[]string{"task"})

}
