package reg

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

// RegDeleteCmd deletes a registry key.
func RegDeleteCmd(cmd *cobra.Command, con *repl.Console) error {
	hive, path, key := common.ParseRegistryFlags(cmd)

	session := con.GetInteractive()
	task, err := RegDelete(con.Rpc, session, hive, path, key)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("delete registry key: %s\\%s\\%s", hive, path, key))
	return nil
}

func RegDelete(rpc clientrpc.MaliceRPCClient, session *core.Session, hive, path, key string) (*clientpb.Task, error) {
	request := &implantpb.RegistryRequest{
		Type: consts.ModuleRegDelete,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: path,
			Key:  key,
		},
	}
	return rpc.RegDelete(session.Context(), request)
}

func RegisterRegDeleteFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleRegDelete,
		RegDelete,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
}
