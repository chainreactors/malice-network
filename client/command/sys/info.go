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

func InfoCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := Info(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("Info error: %v", err)
		return
	}
	session.Console(task, "sysinfo")
}

func Info(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Info(session.Context(), &implantpb.Request{
		Name: consts.ModuleInfo,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
