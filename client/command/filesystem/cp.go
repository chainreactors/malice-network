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

func CpCmd(cmd *cobra.Command, con *repl.Console) error {
	originPath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)

	session := con.GetInteractive()
	task, err := Cp(con.Rpc, session, originPath, targetPath)
	if err != nil {
		return err
	}

	session.Console(cmd, task, "cp "+originPath+" "+targetPath)
	return nil
}

func Cp(rpc clientrpc.MaliceRPCClient, session *core.Session, originPath, targetPath string) (*clientpb.Task, error) {
	task, err := rpc.Cp(session.Context(), &implantpb.Request{
		Name: consts.ModuleCp,
		Args: []string{originPath, targetPath},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
