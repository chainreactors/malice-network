package modules

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ClearCmd(cmd *cobra.Command, con *repl.Console) {
	task, err := clearAll(con.Rpc, con.GetInteractive())
	if err != nil {
		repl.Log.Errorf(err.Error())
		return
	}

	con.AddCallback(task, func(msg proto.Message) {
		repl.Log.Infof("clear all custom modules and exts")
	})
}

func clearAll(rpc clientrpc.MaliceRPCClient, sess *repl.Session) (*clientpb.Task, error) {
	task, err := rpc.Clear(repl.Context(sess), &implantpb.Request{Name: consts.ModuleClear})
	if err != nil {
		return nil, err
	}
	return task, nil
}
