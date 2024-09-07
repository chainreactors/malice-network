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

func CatCmd(cmd *cobra.Command, con *repl.Console) {
	fileName := cmd.Flags().Arg(0)
	if fileName == "" {
		repl.Log.Errorf("required arguments missing")
		return
	}
	task, err := Cat(con.Rpc, con.GetInteractive(), fileName)
	if err != nil {
		repl.Log.Errorf("Cat error: %v", err)
	}
	session := con.GetInteractive()
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		session.Log.Infof("File content: %s", resp.GetOutput())
	})
}

func Cat(rpc clientrpc.MaliceRPCClient, session *repl.Session, fileName string) (*clientpb.Task, error) {
	task, err := rpc.Cat(repl.Context(session), &implantpb.Request{
		Name:  consts.ModuleCat,
		Input: fileName,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
