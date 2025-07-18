package filesystem

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func ChownCmd(cmd *cobra.Command, con *repl.Console) error {
	uid := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)

	gid, _ := cmd.Flags().GetString("gid")
	recursive, _ := cmd.Flags().GetBool("recursive")
	session := con.GetInteractive()
	task, err := Chown(con.Rpc, session, path, uid, gid, recursive)
	if err != nil {
		return err
	}

	session.Console(cmd, task, "chown "+path+" "+uid)
	return nil
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
