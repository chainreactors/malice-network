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

func LsCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	if path == "" {
		path = "./"
	}
	session := con.GetInteractive()
	_, err := Ls(con.Rpc, session, path)
	if err != nil {
		con.Log.Errorf("Ls error: %v", err)
		return
	}
}

func Ls(rpc clientrpc.MaliceRPCClient, session *core.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Ls(session.Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
