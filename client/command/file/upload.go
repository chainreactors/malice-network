package file

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	"os"
)

func UploadCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	target := cmd.Flags().Arg(1)
	priv, _ := cmd.Flags().GetInt("priv")
	hidden, _ := cmd.Flags().GetBool("hidden")

	sid := con.GetInteractive().SessionId
	task, err := Upload(con.Rpc, con.GetInteractive(), path, target, priv, hidden)
	if err != nil {
		return
	}

	con.AddCallback(task.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Consolef("Upload status %v", msg.(*clientpb.Task).GetStatus())
	})
}

func Upload(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, path string, target string, priv int, hidden bool) (*clientpb.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("Can't open file: %s", err)
	}

	task, err := rpc.Upload(console.Context(session), &implantpb.UploadRequest{
		Name:   filepath.Base(path),
		Target: target,
		Priv:   uint32(priv),
		Data:   data,
		Hidden: hidden,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
