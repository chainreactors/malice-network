package sessions

import (
	"context"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func groupCmd(cmd *cobra.Command, con *repl.Console) error {
	sid := cmd.Flags().Arg(1)
	group := cmd.Flags().Arg(0)

	if con.GetInteractive() == nil && sid == "" {
		con.Log.Errorf("No session selected\n")
		return nil
	} else if sid == "" && con.GetInteractive() != nil {
		sid = con.GetInteractive().Session.GetSessionId()
	}

	_, err := con.Rpc.SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sid,
		Op:        "group",
		Arg:       group,
	})
	if err != nil {
		return err
	}
	con.UpdateSession(sid)
	con.Log.Infof("update %s group to %s\n", sid, group)
	return nil
}
