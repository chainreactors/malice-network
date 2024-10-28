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

// RegListKeyCmd lists the keys under a specific registry path.
func RegListKeyCmd(cmd *cobra.Command, con *repl.Console) error {
	hive, path, _ := common.ParseRegistryFlags(cmd)

	session := con.GetInteractive()
	task, err := RegListKey(con.Rpc, session, hive, path)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("list registry keys under: %s\\%s", hive, path))
	return nil
}

func RegListKey(rpc clientrpc.MaliceRPCClient, session *core.Session, hive, path string) (*clientpb.Task, error) {
	request := &implantpb.RegistryRequest{
		Type: consts.ModuleRegListKey,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: path,
		},
	}
	return rpc.RegListKey(session.Context(), request)
}

func RegisterRegListFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleRegListKey,
		RegListKey,
		"",
		nil,
		common.ParseArrayResponse,
		common.FormatArrayResponse,
	)
	con.RegisterImplantFunc(
		consts.ModuleRegListValue,
		RegListValue,
		"",
		nil,
		common.ParseArrayResponse,
		common.FormatArrayResponse,
	)
}

// RegListValueCmd lists the values under a specific registry path.
func RegListValueCmd(cmd *cobra.Command, con *repl.Console) error {
	hive, path, _ := common.ParseRegistryFlags(cmd)

	session := con.GetInteractive()
	task, err := RegListValue(con.Rpc, session, hive, path)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("list registry values under: %s\\%s", hive, path))
	return nil
}

func RegListValue(rpc clientrpc.MaliceRPCClient, session *core.Session, hive, path string) (*clientpb.Task, error) {
	request := &implantpb.RegistryRequest{
		Type: consts.ModuleRegListValue,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: path,
		},
	}
	return rpc.RegListValue(session.Context(), request)
}
