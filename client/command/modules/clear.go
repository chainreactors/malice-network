package modules

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func ClearCmd(cmd *cobra.Command, con *core.Console) error {
	task, err := clearAll(con.Rpc, con.GetInteractive())
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func clearAll(rpc clientrpc.MaliceRPCClient, sess *client.Session) (*clientpb.Task, error) {
	task, err := rpc.Clear(sess.Context(), &implantpb.Request{Name: consts.ModuleClear})
	if err != nil {
		return nil, err
	}
	return task, nil
}
