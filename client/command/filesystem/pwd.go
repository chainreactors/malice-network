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

func PwdCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := Pwd(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("Pwd error: %v", err)
		return
	}
	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		return msg.GetResponse().GetOutput(), nil
	})
}

func Pwd(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Pwd(session.Context(), &implantpb.Request{
		Name: consts.ModulePwd,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
