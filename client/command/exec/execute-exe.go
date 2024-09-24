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

// ExecuteExeCmd - Execute PE on sacrifice process
func ExecuteExeCmd(cmd *cobra.Command, con *repl.Console) {
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	sac, _ := common.ParseSacrificeFlags(cmd)
	task, err := ExecExe(con.Rpc, con.GetInteractive(), path, args, output, timeout, arch, process, sac)
	if err != nil {
		con.Log.Errorf("Execute EXE error: %v", err)
		return
	}
	con.GetInteractive().Console(task, path)
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
func InlineExeCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	path, args, output, timeout, arch, process := common.ParseFullBinaryFlags(cmd)
	task, err := InlineExe(con.Rpc, session, path, args, output, timeout, arch, process)
	if err != nil {
		con.Log.Errorf("Execute EXE error: %v", err)
		return
	}
	session.Console(task, path)
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
