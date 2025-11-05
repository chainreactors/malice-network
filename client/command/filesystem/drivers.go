package filesystem

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func EnumDriverCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := EnumDriver(con.Rpc, session)
	if err != nil {
		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func EnumDriver(rpc clientrpc.MaliceRPCClient, session *client.Session) (*clientpb.Task, error) {
	task, err := rpc.EnumDrivers(session.Context(), &implantpb.Request{
		Name: consts.ModuleEnumDrivers,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
