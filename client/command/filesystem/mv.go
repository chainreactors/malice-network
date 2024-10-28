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

func MvCmd(cmd *cobra.Command, con *repl.Console) error {
	sourcePath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)

	session := con.GetInteractive()
	task, err := Mv(con.Rpc, session, sourcePath, targetPath)
	if err != nil {
		return err
	}

	session.Console(task, "mv "+sourcePath+" "+targetPath)
	return nil
}

func Mv(rpc clientrpc.MaliceRPCClient, session *core.Session, sourcePath string, targetPath string) (*clientpb.Task, error) {
	task, err := rpc.Mv(session.Context(), &implantpb.Request{
		Name: consts.ModuleMv,
		Args: []string{sourcePath, targetPath},
	})
	if err != nil {
		return nil, err
	}
	return task, err

}
