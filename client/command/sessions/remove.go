package sessions

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func removeCmd(cmd *cobra.Command, con *core.Console) error {
	id := cmd.Flags().Arg(0)
	_, err := con.Rpc.SessionManage(con.Context(), &clientpb.BasicUpdateSession{
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
