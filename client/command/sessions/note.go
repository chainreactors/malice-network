package sessions

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func noteCmd(cmd *cobra.Command, con *console.Console) {
	name := cmd.Flags().Arg(0)
	id, err := cmd.Flags().GetString("id")
	if con.GetInteractive().SessionId != "" {
		id = con.GetInteractive().SessionId
	} else if id == "" {
		console.Log.Errorf("Require session id")
		return
	}
	_, err = con.Rpc.BasicSessionOP(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: id,
		Note:      name,
	})
	if err != nil {
		logs.Log.Errorf("Session error: %v", err)
		return
	}
	err = con.UpdateSessions(false)
	if err != nil {
		console.Log.Errorf("update sessions failed: %s", err)
		return
	}
	session := con.Sessions[id]
	con.ActiveTarget.Set(session)
}
