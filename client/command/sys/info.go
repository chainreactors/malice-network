package sys

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func InfoCmd(cmd *cobra.Command, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := session.SessionId
	task, err := Info(con.Rpc, session)
	if err != nil {
		console.Log.Errorf("Info error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Consolef("Info: %v\n", msg.(*implantpb.Spite).Body)
	})
}

func Info(rpc clientrpc.MaliceRPCClient, session *clientpb.Session) (*clientpb.Task, error) {
	task, err := rpc.Info(console.Context(session), &implantpb.Request{
		Name: consts.ModuleInfo,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
