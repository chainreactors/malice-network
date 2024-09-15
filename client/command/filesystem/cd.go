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

func CdCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	if path == "" {
		con.Log.Errorf("required arguments missing")
		return
	}
	task, err := Cd(con.Rpc, con.GetInteractive(), path)
	if err != nil {
		con.Log.Errorf("Cd error: %v", err)
		return
	}

	con.GetInteractive().Console(task, "cd "+path)
}

func Cd(rpc clientrpc.MaliceRPCClient, session *core.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Cd(session.Context(), &implantpb.Request{
		Name:  consts.ModuleCd,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
