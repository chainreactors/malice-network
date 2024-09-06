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

func MvCmd(cmd *cobra.Command, con *repl.Console) {
	sourcePath := cmd.Flags().Arg(0)
	targetPath := cmd.Flags().Arg(1)
	if sourcePath == "" || targetPath == "" {
		repl.Log.Errorf("required arguments missing")
		return
	}

	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := Mv(con.Rpc, session, sourcePath, targetPath)
	if err != nil {
		repl.Log.Errorf("Mv error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Mv success\n")
	})
}

func Mv(rpc clientrpc.MaliceRPCClient, session *repl.Session, sourcePath string, targetPath string) (*clientpb.Task, error) {
	task, err := rpc.Mv(repl.Context(session), &implantpb.Request{
		Name: consts.ModuleMv,
		Args: []string{sourcePath, targetPath},
	})
	if err != nil {
		return nil, err
	}
	return task, err

}
