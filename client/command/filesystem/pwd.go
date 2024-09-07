package filesystem

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func PwdCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := Pwd(con.Rpc, session)
	if err != nil {
		repl.Log.Errorf("Pwd error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		session.Log.Consolef("%s\n", resp.GetOutput())
	})
}

func Pwd(rpc clientrpc.MaliceRPCClient, session *repl.Session) (*clientpb.Task, error) {
	task, err := rpc.Pwd(repl.Context(session), &implantpb.Request{
		Name: consts.ModulePwd,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
