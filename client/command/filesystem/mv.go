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

func MvCmd(cmd *cobra.Command, con *core.Console) error {
	sourcePath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)

	session := con.GetInteractive()
	task, err := Mv(con.Rpc, session, sourcePath, targetPath)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Mv(rpc clientrpc.MaliceRPCClient, session *client.Session, sourcePath, targetPath string) (*clientpb.Task, error) {
	task, err := rpc.Mv(session.Context(), &implantpb.Request{
		Name: consts.ModuleMv,
		Args: []string{sourcePath, targetPath},
	})
	if err != nil {
		return nil, err
	}
	return task, err

}
