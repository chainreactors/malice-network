package exec

import (
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"math"
	"os"
)

func ExecuteDLLCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	sac := common.ParseSacrificeFlags(cmd)
	entrypoint, _ := cmd.Flags().GetString("entrypoint")
	binPath, _ := cmd.Flags().GetString("binPath")
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	task, err := ExecDLL(con.Rpc, session, path, entrypoint, args, binPath, output, timeout, arch, process, sac)
	if err != nil {
		return err
	}
	session.Console(task, path)
	return nil
}

func ExecDLL(rpc clientrpc.MaliceRPCClient, sess *core.Session, dllPath string, entrypoint string, args []string, binPath string, out bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	binary, err := output.NewBinary(consts.ModuleExecuteDll, dllPath, args, out, timeout, arch, process, sac)
	if err != nil {
		return nil, err
	}

	binPath = fileutils.FormatWindowPath(binPath)
	if _, err := os.Stat(binPath); err == nil {
		binData, err := os.ReadFile(binPath)
		if err != nil {
			return nil, err
		}
		binary.Data = binData
	}

	if arch == "" {
		arch = sess.Os.Arch
	}

	binary.EntryPoint = entrypoint
	if pe.CheckPEType(binary.Bin) != consts.DLLFile {
		return nil, errors.New("the file is not a DLL file")
	}
	task, err := rpc.ExecuteEXE(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, err
}

func InlineDLLCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	entryPoint, _ := cmd.Flags().GetString("entrypoint")
	task, err := InlineDLL(con.Rpc, session, path, entryPoint, args, output, timeout, arch, process)
	if err != nil {
		return err
	}
	session.Console(task, path)
	return nil
}

func InlineDLL(rpc clientrpc.MaliceRPCClient, sess *core.Session, path, entryPoint string, args []string,
	out bool, timeout uint32, arch string, process string) (*clientpb.Task, error) {
	if arch == "" {
		arch = sess.Os.Arch
	}
	binary, err := output.NewBinary(consts.ModuleExecuteDll, path, args, out, timeout, arch, process, nil)
	if err != nil {
		return nil, err
	}
	binary.EntryPoint = entryPoint
	if pe.CheckPEType(binary.Bin) != consts.DLLFile {
		return nil, errors.New("the file is not a DLL file")
	}
	task, err := rpc.ExecuteEXE(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterDLLFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleExecuteDll,
		ExecDLL,
		"bdllinject",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, ppid uint32, path string) (*clientpb.Task, error) {
			sac, _ := intermediate.NewSacrificeProcessMessage(ppid, false, true, true, "")
			return ExecDLL(rpc, sess, path, "DLLMain", nil, "", true, math.MaxUint32, sess.Os.Arch, "", sac)
		},
		output.ParseBinaryResponse,
		nil)
	// sess *core.Session, dllPath string, entrypoint string, args []string, binPath string, output bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess
	con.AddCommandFuncHelper(
		consts.ModuleExecuteDll,
		consts.ModuleExecuteDll,
		consts.ModuleExecuteDll+`(active(),"example.dll",{},true,60,"","",new_sacrifice(1234,false,true,true,""))`,
		[]string{
			"session: special session",
			"dllPath",
			"entrypoint",
			"args",
			"binPath",
			"output",
			"timeout",
			"arch",
			"process",
			"sac: sacrifice process",
		},
		[]string{"task"})

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
		output.ParseBinaryResponse,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleAliasInlineDll,
		consts.ModuleAliasInlineDll,
		consts.ModuleAliasInlineDll+`(active(),"example.dll","",{"arg1","arg2"},true,60,"","")`,
		[]string{
			"session: special session",
			"path",
			"entryPoint",
			"args",
			"output",
			"timeout",
			"arch",
			"process",
		},
		[]string{"task"})

}
