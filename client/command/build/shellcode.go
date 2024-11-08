package build

import (
	"context"
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func SRDICmd(cmd *cobra.Command, con *repl.Console) error {
	path, typ, arch, platform, id, functionName, userDataPath := common.ParseSRDIFlags(cmd)
	var fileName string
	var err error
	var bin []byte
	if path == "" && id == "" {
		return errors.New("require path or id")
	} else if path != "" {
		fileName = filepath.Base(path)
		bin, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	}
	resp, err := con.Rpc.MaleficSRDI(context.Background(), &clientpb.MutantFile{
		Id:           id,
		Bin:          bin,
		Arch:         arch,
		Type:         typ,
		Name:         fileName,
		Platform:     platform,
		FunctionName: functionName,
		UserDataPath: userDataPath,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(assets.TempDirName, fileName), resp.Bin, 0644)
	if err != nil {
		return err
	}
	con.Log.Infof("Save mutant file to %s", filepath.Join(assets.TempDirName, fileName))
	return nil
}
