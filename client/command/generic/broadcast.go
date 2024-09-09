package generic

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
	"strings"
)

func BroadcastCmd(cmd *cobra.Command, con *repl.Console) {
	msg := cmd.Flags().Args()
	_, err := Broadcast(con, strings.Join(msg, " "))
	if err != nil {
		con.Log.Errorf("broadcast error: %s", err)
		return
	}
}

func Broadcast(con *repl.Console, msg string) (bool, error) {
	_, err := con.Rpc.Broadcast(con.Context(), &clientpb.Event{
		Type: consts.EventBroadcast,
		Data: []byte(msg),
	})

	return true, err
}
