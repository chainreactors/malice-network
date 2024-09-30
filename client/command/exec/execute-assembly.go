package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func ExecuteAssemblyCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	path, args, output, _ := common.ParseBinaryFlags(cmd)
	//output, _ := cmd.Flags().GetBool("output")
	amsi, etw := common.ParseCLRFlags(cmd)
	task, err := ExecAssembly(con.Rpc, session, path, args, output, amsi, etw)
	if err != nil {
		con.Log.Errorf("Execute error: %v", err)
		return
	}
	con.GetInteractive().Console(task, path)
}

func ExecAssembly(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args []string, output, amsi, etw bool) (*clientpb.Task, error) {
	binary, err := common.NewExecutable(consts.ModuleExecuteAssembly, path, args, sess.Os.Arch, output, nil)
	if err != nil {
		return nil, err
	}
	clr := &implantpb.ExecuteClr{
		AmsiBypass:    amsi,
		EtwBypass:     etw,
		ExecuteBinary: binary,
	}
	task, err := rpc.ExecuteAssembly(sess.Context(), clr)
	if err != nil {
		return nil, err
	}
	return task, nil
}
