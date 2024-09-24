package filesystem

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func CatCmd(cmd *cobra.Command, con *repl.Console) {
	fileName := cmd.Flags().Arg(0)
	if fileName == "" {
		con.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	task, err := Cat(con.Rpc, session, fileName)
	if err != nil {
		con.Log.Errorf("Cat error: %v", err)
	}

	session.Console(task, "cat "+fileName)
}

func Cat(rpc clientrpc.MaliceRPCClient, session *core.Session, fileName string) (*clientpb.Task, error) {
	task, err := rpc.Cat(session.Context(), &implantpb.Request{
		Name:  consts.ModuleCat,
		Input: fileName,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
