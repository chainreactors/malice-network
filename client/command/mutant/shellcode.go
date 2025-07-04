package mutant

import (
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func SRDICmd(cmd *cobra.Command, con *repl.Console) error {
	path, target, id, params := common.ParseSRDIFlags(cmd)
	var err error

	resp, err := MaleficSRDI(con, path, id, target, params)
	if err != nil {
		return err
	}
	srdiPath := filepath.Join(assets.GetTempDir(), resp.Name)
	err = os.WriteFile(srdiPath, resp.Bin, 0644)
	if err != nil {
		return err
	}
	con.Log.Infof("Save mutant file to %s\n", filepath.Join(assets.GetTempDir(), resp.Name))
	return nil
}

func MaleficSRDI(con *repl.Console, path string, id uint32, target string, params map[string]string) (*clientpb.Artifact, error) {
	if path == "" && id == 0 {
		return nil, errors.New("require path or id")
	}
	var bin []byte
	var err error
	if path != "" {
		bin, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}
	return con.Rpc.MaleficSRDI(con.Context(), &clientpb.Artifact{
		Id:           id,
		Bin:          bin,
		Type:         consts.CommandBuildShellCode,
		Name:         filepath.Base(path),
		Target:       target,
		FunctionName: params["function_name"],
		UserDataPath: params["userdata_path"],
	})
}
