package filesystem

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ChownCmd(cmd *cobra.Command, con *repl.Console) {
	uid := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)
	if uid == "" || path == "" {
		repl.Log.Errorf("required arguments missing")
		return
	}
	gid, _ := cmd.Flags().GetString("gid")
	recursive, _ := cmd.Flags().GetBool("recursive")
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := Chown(con.Rpc, con.GetInteractive(), path, uid, gid, recursive)
	if err != nil {
		repl.Log.Errorf("Chown error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		_ = msg.(*clientpb.Task)
		con.SessionLog(sid).Consolef("Chown success\n")
	})
}

func Chown(rpc clientrpc.MaliceRPCClient, session *repl.Session, path, uid, gid string, recursive bool) (*clientpb.Task, error) {
	task, err := rpc.Chown(repl.Context(session), &implantpb.ChownRequest{
		Path:      path,
		Uid:       uid,
		Gid:       gid,
		Recursive: recursive,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
