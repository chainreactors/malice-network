package modules

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"os"
)

func LoadModuleCmd(cmd *cobra.Command, con *repl.Console) {
	bundle := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)
	session := con.GetInteractive()
	task, err := LoadModule(con.Rpc, session, bundle, path)
	if err != nil {
		con.Log.Errorf("LoadModule error: %v", err)
		return
	}
	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		return fmt.Sprintf("LoadModule: %s success", bundle), nil
	})
}

func LoadModule(rpc clientrpc.MaliceRPCClient, session *core.Session, bundle string, path string) (*clientpb.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	task, err := rpc.LoadModule(session.Context(), &implantpb.LoadModule{
		Bundle: bundle,
		Bin:    data,
	})

	if err != nil {
		return nil, err
	}
	return task, nil
}
