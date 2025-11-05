package basic

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/utils"
	"github.com/spf13/cobra"
)

func SwitchCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	input := cmd.Flags().Args()
	var urls []string
	for _, u := range input {
		if pipe, ok := con.Pipelines[u]; ok {
			urls = append(urls, pipe.Address())
		} else if addr := utils.NewAddr(u); addr != nil {
			urls = append(urls, addr.String())
		} else {
			session.Log.Warnf("no such pipeline or valid address: %s\n", u)
		}
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
