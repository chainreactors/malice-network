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

func CdCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	task, err := Cd(con.Rpc, con.GetInteractive(), path)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Cd(rpc clientrpc.MaliceRPCClient, session *client.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Cd(session.Context(), &implantpb.Request{
		Name:  consts.ModuleCd,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
