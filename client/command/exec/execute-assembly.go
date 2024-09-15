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
	"github.com/spf13/cobra"
)

func ExecuteAssemblyCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	path, args, output, _ := common.ParseBinaryParams(cmd)
	//output, _ := cmd.Flags().GetBool("output")
	task, err := ExecAssembly(con.Rpc, session, path, args, output)
	if err != nil {
		con.Log.Errorf("Execute error: %v", err)
		return
	}
	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		resp, _ := builtin.ParseAssembly(msg)
		return resp, nil
	})
}

func ExecAssembly(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args []string, output bool) (*clientpb.Task, error) {
	binary, err := common.NewExecutable(consts.ModuleExecuteAssembly, path, args, sess.Os.Arch, output, nil)
	if err != nil {
		return nil, err
	}
	task, err := rpc.ExecuteAssembly(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}
