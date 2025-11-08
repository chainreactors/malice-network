package filesystem

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func CatCmd(cmd *cobra.Command, con *repl.Console) error {
	fileName := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := Cat(con.Rpc, session, fileName)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Cat(rpc clientrpc.MaliceRPCClient, session *client.Session, fileName string) (*clientpb.Task, error) {
	task, err := rpc.Cat(session.Context(), &implantpb.Request{
		Name:  consts.ModuleCat,
		Input: fileName,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
