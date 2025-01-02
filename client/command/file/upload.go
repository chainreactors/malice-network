package file

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/mals"
	"github.com/spf13/cobra"

	"os"
)

func UploadCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	target := cmd.Flags().Arg(1)
	priv, _ := cmd.Flags().GetString("priv")
	hidden, _ := cmd.Flags().GetBool("hidden")

	task, err := Upload(con.Rpc, con.GetInteractive(), path, target, priv, hidden)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, fmt.Sprintf("Upload %s", path))
	return nil
}

func Upload(rpc clientrpc.MaliceRPCClient, session *core.Session, path string, target string, priv string, hidden bool) (*clientpb.Task, error) {
	var data []byte
	var err error
	// data, err := os.ReadFile(path)
	if fileutils.Exist(path) {
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("ReadFile error: %s", err)
		}
	} else {
		data, err = mals.UnPackMalBinary(path)
		if err != nil {
			return nil, fmt.Errorf("the path does not point to a valid file or does not meet the expected binary format: %s", err)
		}
		path = "virtual_src_path"
	}

	if err != nil {
		return nil, err
	}
	value, err := strconv.ParseUint(priv, 8, 32)
	if err != nil {
		return nil, err
	}
	task, err := rpc.Upload(session.Context(), &implantpb.UploadRequest{
		Name:   filepath.Base(path),
		Target: target,
		Priv:   uint32(value),
		Data:   data,
		Hidden: hidden,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
