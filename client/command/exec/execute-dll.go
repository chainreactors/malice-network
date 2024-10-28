package exec

import (
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
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

func ExecuteDLLCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	sac, _ := common.ParseSacrificeFlags(cmd)
	entrypoint, _ := cmd.Flags().GetString("entrypoint")
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	task, err := ExecDLL(con.Rpc, session, path, entrypoint, args, output, timeout, arch, process, sac)
	if err != nil {
		return err
	}
	session.Console(task, path)
	return nil
}

func ExecDLL(rpc clientrpc.MaliceRPCClient, sess *core.Session, pePath string, entrypoint string, args []string, output bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	binary, err := common.NewBinary(consts.ModuleExecuteDll, pePath, args, output, timeout, arch, process, sac)
	if err != nil {
		return nil, err
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
	output bool, timeout uint32, arch string, process string) (*clientpb.Task, error) {
	if arch == "" {
		arch = sess.Os.Arch
	}
	binary, err := common.NewBinary(consts.ModuleExecuteDll, path, args, output, timeout, arch, process, nil)
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
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, ppid int, path string) (*clientpb.Task, error) {
			sac, _ := intermediate.NewSacrificeProcessMessage(int64(ppid), false, true, true, "")
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

}
