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

func ChmodCmd(cmd *cobra.Command, con *console.Console) {
	mode := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)
	if mode == "" || path == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := Chmod(con.Rpc, con.GetInteractive(), path, mode)
	if err != nil {
		console.Log.Errorf("Chmod error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		_ = msg.(*clientpb.Task)
		con.SessionLog(sid).Consolef("Chmod success\n")
	})
}

func Chmod(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, path, mode string) (*clientpb.Task, error) {
	task, err := rpc.Chmod(console.Context(session), &implantpb.Request{
		Name: consts.ModuleChmod,
		Args: []string{path, mode},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
