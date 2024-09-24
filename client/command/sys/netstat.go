package sys

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func NetstatCmd(cmd *cobra.Command, con *repl.Console) {
	task, err := Netstat(con.Rpc, con.GetInteractive())
	if err != nil {
		con.Log.Errorf("Kill error: %v", err)
		return
	}
	con.GetInteractive().Console(task, "netstat")
}

func Netstat(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Netstat(session.Context(), &implantpb.Request{
		Name: consts.ModuleNetstat,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
