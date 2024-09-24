package sessions

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func groupCmd(cmd *cobra.Command, con *repl.Console) {
	sid := cmd.Flags().Arg(1)
	group := cmd.Flags().Arg(0)

	if con.GetInteractive() == nil && sid == "" {
		con.Log.Errorf("No session selected")
		return
	} else if sid == "" && con.GetInteractive() != nil {
		sid = con.GetInteractive().Session.GetSessionId()
	}

	_, err := con.Rpc.BasicSessionOP(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sid,
		Op:        "group",
		Arg:       group,
	})
	if err != nil {
		logs.Log.Errorf("Session error: %v", err)
		return
	}
	con.UpdateSession(sid)
	con.Log.Infof("update %s group to %s", sid, group)
}
