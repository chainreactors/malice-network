package sessions

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func groupCmd(cmd *cobra.Command, con *core.Console) error {
	sid, err := resolveSessionID(con, cmd.Flags().Arg(1))
	group := cmd.Flags().Arg(0)
	if err != nil {
		return err
	}

	_, err = con.Rpc.SessionManage(con.Context(), &clientpb.BasicUpdateSession{
		SessionId: sid,
		Op:        "group",
		Arg:       group,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("update %s group to %s\n", sid, group)
	return nil
}
