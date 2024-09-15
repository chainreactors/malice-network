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

func ExecuteBofCmd(cmd *cobra.Command, con *repl.Console) {
	path, args, output, _ := common.ParseBinaryParams(cmd)
	task, err := ExecBof(con.Rpc, con.GetInteractive(), path, args, output)
	if err != nil {
		con.Log.Errorf("Execute BOF error: %v", err)
		return
	}

	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		resp, _ := builtin.ParseAssembly(msg)
		return resp, nil
	})
}

func ExecBof(rpc clientrpc.MaliceRPCClient, sess *core.Session, bofPath string, args []string, output bool) (*clientpb.Task, error) {
	binary, err := common.NewExecutable(consts.ModuleExecuteBof, bofPath, args, sess.Os.Arch, output, nil)
	if err != nil {
		return nil, err
	}
	task, err := rpc.ExecuteBof(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}
