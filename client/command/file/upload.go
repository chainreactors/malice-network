package file

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
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
	data, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("Can't open file: %s", err)
	}
	upload(path, data, target, priv, hidden, con)
}

func upload(path string, data []byte, target string, priv int, hidden bool, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	uploadTask, err := con.Rpc.Upload(con.ActiveTarget.Context(), &implantpb.UploadRequest{
		Name:   filepath.Base(path),
		Target: target,
		Priv:   uint32(priv),
		Data:   data,
		Hidden: hidden,
	})
	if err != nil {
		console.Log.Errorf("Upload error: %v", err)
		return
	}
	con.AddCallback(uploadTask.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Consolef("upload status %v", msg.(*clientpb.Task).GetStatus())
	})
}
