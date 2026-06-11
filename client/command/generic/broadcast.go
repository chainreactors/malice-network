package generic

import (
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"strings"
)

func BroadcastCmd(cmd *cobra.Command, con *core.Console) error {
	msg := cmd.Flags().Args()
	isNotify, _ := cmd.Flags().GetBool("notify")
	if isNotify {
		_, err := Notify(con, &clientpb.Event{
			Type:    consts.EventNotify,
			Client:  con.Client,
			Message: []byte(strings.Join(msg, " ")),
		})
		return err
	}

	_, err := Broadcast(con, &clientpb.Event{
		Type:    consts.EventBroadcast,
		Client:  con.Client,
		Message: []byte(strings.Join(msg, " ")),
	})
	return err
}

func Broadcast(con *core.Console, event *clientpb.Event) (bool, error) {
	_, err := con.Rpc.Broadcast(con.Context(), event)
	if err != nil {
		return false, err
	}
	return true, nil
}

func Notify(con *core.Console, event *clientpb.Event) (bool, error) {
	_, err := con.Rpc.Notify(con.Context(), event)
	if err != nil {
		return false, err
	}
	return true, nil
}
