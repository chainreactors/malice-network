package sessions

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func removeCmd(cmd *cobra.Command, con *repl.Console) {
	id := cmd.Flags().Arg(0)
	_, err := con.Rpc.BasicSessionOP(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: id,
		Op:        "delete",
	})
	if err != nil {
		logs.Log.Errorf("Session error: %v", err)
		return
	}
	delete(con.Sessions, id)
	repl.Log.Infof("delete session %s", id)
}
