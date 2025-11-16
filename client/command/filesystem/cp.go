package filesystem

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func CpCmd(cmd *cobra.Command, con *core.Console) error {
	originPath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)

	session := con.GetInteractive()
	task, err := Cp(con.Rpc, session, originPath, targetPath)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Cp(rpc clientrpc.MaliceRPCClient, session *client.Session, originPath, targetPath string) (*clientpb.Task, error) {
	task, err := rpc.Cp(session.Context(), &implantpb.Request{
		Name: consts.ModuleCp,
		Args: []string{originPath, targetPath},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
