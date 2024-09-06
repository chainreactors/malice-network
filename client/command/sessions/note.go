package sessions

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func noteCmd(cmd *cobra.Command, con *console.Console) {
	sid := cmd.Flags().Arg(0)
	name := cmd.Flags().Arg(1)

	var err error
	_, err = con.Rpc.BasicSessionOP(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sid,
		Op:        "note",
		Arg:       name,
	})
	if err != nil {
		logs.Log.Errorf("Session error: %v", err)
		return
	}
	con.UpdateSession(sid)
	console.Log.Infof("update %s note to %s", sid, name)
}
