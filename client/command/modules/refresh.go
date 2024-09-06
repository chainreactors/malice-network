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

func RefreshModuleCmd(cmd *cobra.Command, con *repl.Console) {
	task, err := refreshModule(con.Rpc, con.GetInteractive())
	if err != nil {
		repl.Log.Errorf(err.Error())
		return
	}

	con.AddCallback(task, func(msg proto.Message) {
		repl.Log.Infof("Module refreshed")
	})
}

func refreshModule(rpc clientrpc.MaliceRPCClient, session *repl.Session) (*clientpb.Task, error) {
	task, err := rpc.RefreshModule(repl.Context(session), &implantpb.Request{Name: consts.ModuleRefreshModule})
	if err != nil {
		return nil, err
	}
	return task, nil
}
