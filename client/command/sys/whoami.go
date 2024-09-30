package sys

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func WhoamiCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := Whoami(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("Whoami error: %v", err)
		return
	}
	session.Console(task, "")
}

func Whoami(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Whoami(session.Context(), &implantpb.Request{
		Name: consts.ModuleWhoami,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
