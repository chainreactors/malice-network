package sessions

import (
	"context"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func removeCmd(cmd *cobra.Command, con *repl.Console) error {
	id := cmd.Flags().Arg(0)
	_, err := con.Rpc.SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: id,
		Op:        "delete",
	})
	if err != nil {
		return err
	}
	delete(con.Sessions, id)
	con.Log.Infof("delete session %s\n", id)
	return nil
}
