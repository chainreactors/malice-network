package file

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"path"
	"strings"
)

func DownloadCmd(cmd *cobra.Command, con *core.Console) error {
	path := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	is_dir, _ := cmd.Flags().GetBool("dir")
	task, err := Download(con.Rpc, session, path, is_dir)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

// remotePath extracts the basename from a remote path that may use
// either forward slashes (Unix) or backslashes (Windows).
func remotePath(p string) string {
	// Normalise Windows backslashes so path.Base works on all platforms.
	return path.Base(strings.ReplaceAll(p, `\`, "/"))
}

func Download(rpc clientrpc.MaliceRPCClient, session *client.Session, path string, is_dir bool) (*clientpb.Task, error) {
	task, err := rpc.Download(session.Context(), &implantpb.DownloadRequest{
		Name: remotePath(path),
		Path: path,
		Dir:  is_dir,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
