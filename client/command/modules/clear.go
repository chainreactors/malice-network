package modules

import (
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/session"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func ClearCmd(cmd *cobra.Command, con *repl.Console) error {
	task, err := clearAll(con.Rpc, con.GetInteractive())
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func clearAll(rpc clientrpc.MaliceRPCClient, sess *session.Session) (*clientpb.Task, error) {
	task, err := rpc.Clear(sess.Context(), &implantpb.Request{Name: consts.ModuleClear})
	if err != nil {
		return nil, err
	}
	return task, nil
}
