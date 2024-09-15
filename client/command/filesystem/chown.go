package filesystem

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func ChownCmd(cmd *cobra.Command, con *repl.Console) {
	uid := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)
	if uid == "" || path == "" {
		con.Log.Errorf("required arguments missing")
		return
	}
	gid, _ := cmd.Flags().GetString("gid")
	recursive, _ := cmd.Flags().GetBool("recursive")
	session := con.GetInteractive()
	task, err := Chown(con.Rpc, session, path, uid, gid, recursive)
	if err != nil {
		con.Log.Errorf("Chown error: %v", err)
		return
	}
	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		return "Chown success", nil
	})
}

func Chown(rpc clientrpc.MaliceRPCClient, session *core.Session, path, uid, gid string, recursive bool) (*clientpb.Task, error) {
	task, err := rpc.Chown(session.Context(), &implantpb.ChownRequest{
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
