package build

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
	path, arch, platform, id, params := common.ParseSRDIFlags(cmd)
	var fileName string
	var err error

	resp, err := MaleficSRDI(con, path, id, arch, platform, params)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(assets.TempDirName, resp.Name), resp.Bin, 0644)
	if err != nil {
		return err
	}
	con.Log.Infof("Save mutant file to %s", filepath.Join(assets.TempDirName, fileName))
	return nil
}

func MaleficSRDI(con *repl.Console, path string, id uint32, arch, platform string, params map[string]string) (*clientpb.Builder, error) {
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
	return con.Rpc.MaleficSRDI(con.Context(), &clientpb.Builder{
		Id:           id,
		Bin:          bin,
		Arch:         arch,
		Type:         consts.SRDIType,
		Name:         filepath.Base(path),
		Platform:     platform,
		FunctionName: params["function_name"],
		UserDataPath: params["userdata_path"],
	})
}
