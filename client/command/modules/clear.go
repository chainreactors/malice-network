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

func ClearCmd(cmd *cobra.Command, con *repl.Console) error {
	task, err := clearAll(con.Rpc, con.GetInteractive())
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, "clear all custom modules and exts")
	return nil
}

func clearAll(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Clear(sess.Context(), &implantpb.Request{Name: consts.ModuleClear})
	if err != nil {
		return nil, err
	}
	return task, nil
}
