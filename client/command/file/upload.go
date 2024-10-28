package file

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"path/filepath"

	"os"
)

func UploadCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	target := cmd.Flags().Arg(1)
	priv, _ := cmd.Flags().GetInt("priv")
	hidden, _ := cmd.Flags().GetBool("hidden")

	task, err := Upload(con.Rpc, con.GetInteractive(), path, target, priv, hidden)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, fmt.Sprintf("Upload %s", path))
	return nil
}

func Upload(rpc clientrpc.MaliceRPCClient, session *core.Session, path string, target string, priv int, hidden bool) (*clientpb.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		core.Log.Errorf("Can't open file: %s", err)
	}

	task, err := rpc.Upload(session.Context(), &implantpb.UploadRequest{
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
