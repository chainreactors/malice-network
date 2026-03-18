package sessions

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func noteCmd(cmd *cobra.Command, con *core.Console) error {
	sid, err := resolveSessionID(con, cmd.Flags().Arg(1))
	name := cmd.Flags().Arg(0)
	if err != nil {
		return err
	}

	_, err = con.Rpc.SessionManage(con.Context(), &clientpb.BasicUpdateSession{
		SessionId: sid,
		Op:        "note",
		Arg:       name,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("update %s note to %s\n", sid, name)
	return nil
}
