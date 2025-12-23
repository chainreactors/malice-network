package basic

import (
	"fmt"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/utils"
	"github.com/spf13/cobra"
)

func SwitchCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	pipeline, _ := cmd.Flags().GetString("pipeline")
	address, _ := cmd.Flags().GetString("address")

	var urls []string
	if pipeline != "" {
		if pipe, ok := con.Pipelines[pipeline]; ok {
			urls = append(urls, pipe.Address())
		} else {
			return fmt.Errorf("no such pipeline: %s", pipeline)
		}
	}
	if address != "" {
		if addr := utils.NewAddr(address); addr != nil {
			urls = append(urls, addr.String())
		} else {
			return fmt.Errorf("invalid address format: %s", address)
		}
	}

	if len(urls) == 0 {
		return fmt.Errorf("must specify --pipeline or --address")
	}

	task, err := Switch(con.Rpc, session, urls)
	if err != nil {
		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Switch(rpc clientrpc.MaliceRPCClient, session *client.Session, urls []string) (*clientpb.Task, error) {
	return rpc.Switch(session.Context(), &implantpb.Switch{Urls: urls})
}
