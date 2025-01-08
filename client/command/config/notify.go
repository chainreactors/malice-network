package config

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func GetNotifyCmd(cmd *cobra.Command, con *repl.Console) error {
	notifyConfig, err := con.Rpc.GetNotifyConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	con.Log.Console(tui.RendStructDefault(notifyConfig) + "\n")
	return nil
}

func UpdateNotifyCmd(cmd *cobra.Command, con *repl.Console) error {
	notify := common.ParseNotifyFlags(cmd)
	_, err := UpdateNotify(con, notify)
	if err != nil {
		return err
	}
	con.Log.Console("Update notify config success\n")
	return nil
}

func UpdateNotify(con *repl.Console, notify *clientpb.Notify) (*clientpb.Empty, error) {
	return con.Rpc.UpdateNotifyConfig(con.Context(), notify)
}
