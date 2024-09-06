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

func RmCmd(cmd *cobra.Command, con *repl.Console) {
	fileName := cmd.Flags().Arg(0)
	if fileName == "" {
		repl.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := Rm(con.Rpc, session, fileName)
	if err != nil {
		repl.Log.Errorf("Rm error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Removed file success\n")
	})
}

func Rm(rpc clientrpc.MaliceRPCClient, session *repl.Session, fileName string) (*clientpb.Task, error) {
	task, err := rpc.Rm(repl.Context(session), &implantpb.Request{
		Name:  consts.ModuleRm,
		Input: fileName,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
