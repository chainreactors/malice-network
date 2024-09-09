package sys

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func WhoamiCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := Whoami(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("Whoami error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		session.Log.Consolef("%v\n", resp.GetOutput())
	})
}

func Whoami(rpc clientrpc.MaliceRPCClient, session *repl.Session) (*clientpb.Task, error) {
	task, err := rpc.Whoami(session.Context(), &implantpb.Request{
		Name: consts.ModuleWhoami,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
