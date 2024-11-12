package generic

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"strings"
)

func BroadcastCmd(cmd *cobra.Command, con *repl.Console) {
	msg := cmd.Flags().Args()
	isNotify, _ := cmd.Flags().GetBool("notify")
	var err error
	if isNotify {
		_, err = Notify(con, &clientpb.Event{
			Type:    consts.EventNotify,
			Client:  con.Client,
			Message: []byte(strings.Join(msg, " ")),
		})
		if err != nil {
			con.Log.Errorf("notify error: %s\n", err)
			return
		}
	} else {
		_, err = Broadcast(con, &clientpb.Event{
			Type:    consts.EventBroadcast,
			Client:  con.Client,
			Message: []byte(strings.Join(msg, " ")),
		})
		if err != nil {
			con.Log.Errorf("broadcast error: %s\n", err)
			return
		}
	}
}

func Broadcast(con *repl.Console, event *clientpb.Event) (bool, error) {
	_, err := con.Rpc.Broadcast(con.Context(), event)
	return true, err
}

func Notify(con *repl.Console, event *clientpb.Event) (bool, error) {
	_, err := con.Rpc.Notify(con.Context(), event)
	return true, err
}
