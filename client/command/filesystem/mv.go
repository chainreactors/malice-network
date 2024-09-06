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

func MvCmd(cmd *cobra.Command, con *console.Console) {
	sourcePath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)
	if sourcePath == "" || targetPath == "" {
		console.Log.Errorf("required arguments missing")
		return
	}

	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := Mv(con.Rpc, session, sourcePath, targetPath)
	if err != nil {
		console.Log.Errorf("Mv error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Mv success\n")
	})
}

func Mv(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, sourcePath string, targetPath string) (*clientpb.Task, error) {
	task, err := rpc.Mv(console.Context(session), &implantpb.Request{
		Name: consts.ModuleMv,
		Args: []string{sourcePath, targetPath},
	})
	if err != nil {
		return nil, err
	}
	return task, err

}
