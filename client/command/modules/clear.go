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

func ClearCmd(cmd *cobra.Command, con *repl.Console) {
	task, err := clearAll(con.Rpc, con.GetInteractive())
	if err != nil {
		con.Log.Errorf(err.Error())
		return
	}

	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		return "clear all custom modules and exts", nil
	})
}

func clearAll(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Clear(sess.Context(), &implantpb.Request{Name: consts.ModuleClear})
	if err != nil {
		return nil, err
	}
	return task, nil
}
