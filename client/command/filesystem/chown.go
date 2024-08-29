package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ChownCmd(cmd *cobra.Command, con *console.Console) {
	uid := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)
	if uid == "" || path == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	gid, _ := cmd.Flags().GetString("gid")
	recursive, _ := cmd.Flags().GetBool("recursive")
	chown(path, uid, gid, recursive, con)
}

func chown(path string, uid string, gid string, recursive bool, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId

	chownTask, err := con.Rpc.Chown(con.ActiveTarget.Context(), &implantpb.ChownRequest{
		Path:      path,
		Uid:       uid,
		Gid:       gid,
		Recursive: recursive,
	})
	if err != nil {
		console.Log.Errorf("Chown error: %v", err)
		return
	}
	con.AddCallback(chownTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Response)
		con.SessionLog(sid).Consolef("Chown success\n")
	})
}
