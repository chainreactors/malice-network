package config

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func GetNotifyCmd(cmd *cobra.Command, con *core.Console) error {
	notifyConfig, err := con.Rpc.GetNotifyConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	con.Log.Console(tui.RendStructDefault(notifyConfig) + "\n")
	return nil
}

func UpdateNotifyCmd(cmd *cobra.Command, con *core.Console) error {
	current, err := con.Rpc.GetNotifyConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}

	notify := mergeNotifyUpdate(current, cmd)
	_, err = UpdateNotify(con, notify)
	if err != nil {
		return err
	}
	con.Log.Console("Update notify config success\n")
	return nil
}

func UpdateNotify(con *core.Console, notify *clientpb.Notify) (*clientpb.Empty, error) {
	return con.Rpc.UpdateNotifyConfig(con.Context(), notify)
}
