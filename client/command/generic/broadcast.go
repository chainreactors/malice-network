package generic

import (
	"context"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"strings"
)

func BroadcastCmd(cmd *cobra.Command, con *repl.Console) {
	msg := cmd.Flags().Args()
	err := Broadcast(con.Rpc, strings.Join(msg, " "))
	if err != nil {
		con.Log.Errorf("broadcast error: %s", err)
		return
	}
}

func Broadcast(rpc clientrpc.MaliceRPCClient, msg string) error {
	_, err := rpc.Broadcast(context.Background(), &clientpb.Event{
		Type: consts.EventBroadcast,
		Data: []byte(msg),
	})
	return err
}
