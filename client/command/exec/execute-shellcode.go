package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// ExecuteShellcodeCmd - Execute shellcode in-memory
func ExecuteShellcodeCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	path, args, output, timeout, arch, process := common.ParseFullBinaryParams(cmd)
	sac, _ := common.ParseSacrifice(cmd)
	task, err := ExecShellcode(con.Rpc, session, path, args, output, timeout, arch, process, sac)
	if err != nil {
		con.Log.Errorf("Execute shellcode error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp, _ := builtin.ParseAssembly(msg.(*implantpb.Spite))
		session.Log.Console(resp)
	})

}

func ExecShellcode(rpc clientrpc.MaliceRPCClient, sess *repl.Session, shellcodePath string,
	args []string, output bool, timeout int, arch string, process string,
	sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	if arch == "" {
		arch = sess.Os.Arch
	}
	shellcodeBin, err := common.NewBinary(consts.ModuleExecuteShellcode, shellcodePath, args, output, timeout, arch, process, sac)
	task, err := rpc.ExecuteShellcode(sess.Context(), shellcodeBin)
	if err != nil {
		return nil, err
	}
	return task, err
}

func InlineShellcodeCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	path, args, output, timeout, arch, process := common.ParseFullBinaryParams(cmd)
	task, err := InlineShellcode(con.Rpc, session, path, args, output, timeout, arch, process)
	if err != nil {
		con.Log.Errorf("Execute inline shellcode error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp, _ := builtin.ParseAssembly(msg.(*implantpb.Spite))
		session.Log.Console(resp)
	})

}

func InlineShellcode(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string, args []string,
	output bool, timeout int, arch string, process string) (*clientpb.Task, error) {
	if arch == "" {
		arch = sess.Os.Arch
	}
	binary, err := common.NewBinary(consts.ModuleExecuteShellcode, path, args, output, timeout, arch, process, nil)
	if err != nil {
		return nil, err
	}
	shellcodeTask, err := rpc.ExecuteShellcode(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return shellcodeTask, err
}
