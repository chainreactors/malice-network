package exec

import (
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func ExecuteDLLCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	sac, _ := common.ParseSacrifice(cmd)
	entrypoint, _ := cmd.Flags().GetString("entrypoint")
	path, args, output, timeout, arch, process := common.ParseFullBinaryParams(cmd)
	task, err := ExecDLL(con.Rpc, session, path, entrypoint, args, output, timeout, arch, process, sac)
	if err != nil {
		con.Log.Errorf("Execute DLL error: %v", err)
		return
	}
	session.Console(task, path)
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

func InlineDLLCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	path, args, output, timeout, arch, process := common.ParseFullBinaryParams(cmd)
	entryPoint, _ := cmd.Flags().GetString("entrypoint")
	task, err := InlineDLL(con.Rpc, session, path, entryPoint, args, output, timeout, arch, process)
	if err != nil {
		con.Log.Errorf("Execute Inline DLL error: %s", err)
		return
	}
	session.Console(task, path)
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
