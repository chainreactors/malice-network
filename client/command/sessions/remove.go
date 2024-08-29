package sessions

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func removeCmd(cmd *cobra.Command, con *console.Console) {
	id := cmd.Flags().Arg(0)
	if con.GetInteractive().SessionId != "" {
		id = con.GetInteractive().SessionId
	} else if id == "" {
		console.Log.Errorf("Require session id")
		return
	}
	_, err := con.Rpc.BasicSessionOP(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: id,
		IsDelete:  true,
	})
	if err != nil {
		logs.Log.Errorf("Session error: %v", err)
		return
	}
	err = con.UpdateSessions(false)
	if err != nil {
		console.Log.Errorf("%s", err)
		return
	}
	session := con.Sessions[id]
	con.ActiveTarget.Set(session)
}
