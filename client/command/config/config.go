package config

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func RefreshCmd(cmd *cobra.Command, con *repl.Console) error {
	isClient, _ := cmd.Flags().GetBool("client")
	if isClient {
		err := assets.RefreshProfile()
		if err != nil {
			return err
		}
		con.Log.Console("Refresh client config success\n")
		return nil
	} else {
		_, err := Refresh(con)
		if err != nil {
			return err
		}
		con.Log.Console("Refresh config success\n")
		return nil
	}
}

func Refresh(con *repl.Console) (*clientpb.Empty, error) {
	return con.Rpc.RefreshConfig(con.Context(), &clientpb.Empty{})
}
