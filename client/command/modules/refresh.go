package modules

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func RefreshModuleCmd(cmd *cobra.Command, con *repl.Console) error {
	task, err := refreshModule(con.Rpc, con.GetInteractive())
	if err != nil {
		return err
	}

	con.GetInteractive().Console(cmd, task, "refresh module")
	return nil
}

func refreshModule(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.RefreshModule(session.Context(), &implantpb.Request{Name: consts.ModuleRefreshModule})
	if err != nil {
		return nil, err
	}
	return task, nil
}
