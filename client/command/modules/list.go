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

func ListModulesCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := ListModules(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("ListModules error: %v", err)
		return
	}
	session.Console(task, "list modules")
}

func ListModules(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	listTask, err := rpc.ListModule(session.Context(), &implantpb.Request{Name: consts.ModuleListModule})
	if err != nil {
		return nil, err
	}
	return listTask, nil
}
