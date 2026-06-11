package modules

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func UnloadModuleCmd(cmd *cobra.Command, con *core.Console) error {
	bundleName := cmd.Flags().Args()[0]
	session := con.GetInteractive()
	task, err := unloadModule(con.Rpc, session, bundleName)
	if err != nil {
		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func unloadModule(rpc clientrpc.MaliceRPCClient, session *client.Session, bundle string) (*clientpb.Task, error) {
	return rpc.UnloadModule(session.Context(), &implantpb.Request{
		Name:  consts.ModuleUnloadModule,
		Input: bundle,
	})
}
