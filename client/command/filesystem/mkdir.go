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

func MkdirCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	if path == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := Mkdir(con.Rpc, session, path)
	if err != nil {
		console.Log.Errorf("Mkdir error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Created directory\n")
	})
}

func Mkdir(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Mkdir(console.Context(session), &implantpb.Request{
		Name:  consts.ModuleMkdir,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
