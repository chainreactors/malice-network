package filesystem

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func CatCmd(cmd *cobra.Command, con *repl.Console) error {
	fileName := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := Cat(con.Rpc, session, fileName)
	if err != nil {
		return err
	}

	session.Console(cmd, task, "cat "+fileName)
	return nil
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
