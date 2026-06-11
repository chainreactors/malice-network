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

func TouchCmd(cmd *cobra.Command, con *core.Console) error {
	path := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := Touch(con.Rpc, session, path)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Touch(rpc clientrpc.MaliceRPCClient, session *client.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Touch(session.Context(), &implantpb.Request{
		Name:  consts.ModuleTouch,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
