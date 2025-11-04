package filesystem

import (
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/session"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func ChmodCmd(cmd *cobra.Command, con *repl.Console) error {
	mode := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)

	task, err := Chmod(con.Rpc, con.GetInteractive(), path, mode)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return err
}

func Chmod(rpc clientrpc.MaliceRPCClient, session *session.Session, path, mode string) (*clientpb.Task, error) {
	task, err := rpc.Chmod(session.Context(), &implantpb.Request{
		Name: consts.ModuleChmod,
		Args: []string{path, mode},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
