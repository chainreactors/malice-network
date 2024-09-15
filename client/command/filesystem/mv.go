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

func MvCmd(cmd *cobra.Command, con *repl.Console) {
	sourcePath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)
	if sourcePath == "" || targetPath == "" {
		con.Log.Errorf("required arguments missing")
		return
	}

	session := con.GetInteractive()
	task, err := Mv(con.Rpc, session, sourcePath, targetPath)
	if err != nil {
		con.Log.Errorf("Mv error: %v", err)
		return
	}

	session.Console(task, "mv "+sourcePath+" "+targetPath)
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
