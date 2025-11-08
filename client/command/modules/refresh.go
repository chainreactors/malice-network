package modules

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func RefreshModuleCmd(cmd *cobra.Command, con *repl.Console) error {
	task, err := refreshModule(con.Rpc, con.GetInteractive())
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func refreshModule(rpc clientrpc.MaliceRPCClient, session *client.Session) (*clientpb.Task, error) {
	task, err := rpc.RefreshModule(session.Context(), &implantpb.Request{Name: consts.ModuleRefreshModule})
	if err != nil {
		return nil, err
	}
	return task, nil
}
