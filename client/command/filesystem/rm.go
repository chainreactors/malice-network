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

func RmCmd(cmd *cobra.Command, con *repl.Console) error {
	fileName := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := Rm(con.Rpc, session, fileName)
	if err != nil {
		return err
	}

	session.Console(cmd, task, "rm "+fileName)
	return nil
}

func Rm(rpc clientrpc.MaliceRPCClient, session *core.Session, fileName string) (*clientpb.Task, error) {
	task, err := rpc.Rm(session.Context(), &implantpb.Request{
		Name:  consts.ModuleRm,
		Input: fileName,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
