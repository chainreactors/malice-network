package modules

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func RefreshModuleCmd(cmd *cobra.Command, con *repl.Console) {
	task, err := refreshModule(con.Rpc, con.GetInteractive())
	if err != nil {
		con.Log.Errorf(err.Error())
		return
	}

	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		return "Module refreshed", nil
	})
}

func refreshModule(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.RefreshModule(session.Context(), &implantpb.Request{Name: consts.ModuleRefreshModule})
	if err != nil {
		return nil, err
	}
	return task, nil
}
