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

func MkdirCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	if path == "" {
		con.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	task, err := Mkdir(con.Rpc, session, path)
	if err != nil {
		con.Log.Errorf("Mkdir error: %v", err)
		return
	}

	session.Console(task, "mkdir "+path)
}

func Mkdir(rpc clientrpc.MaliceRPCClient, session *core.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Mkdir(session.Context(), &implantpb.Request{
		Name:  consts.ModuleMkdir,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
