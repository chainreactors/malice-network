package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func PwdCmd(cmd *cobra.Command, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	pwdTask, err := Pwd(con.Rpc, session)
	if err != nil {
		console.Log.Errorf("Pwd error: %v", err)
		return
	}
	con.AddCallback(pwdTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetResponse()
		con.SessionLog(sid).Consolef("%s\n", resp.GetOutput())
	})
}

func Pwd(rpc clientrpc.MaliceRPCClient, session *clientpb.Session) (*clientpb.Task, error) {
	task, err := rpc.Pwd(console.Context(session), &implantpb.Request{
		Name: consts.ModulePwd,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
