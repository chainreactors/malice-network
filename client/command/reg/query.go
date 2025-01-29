package reg

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// RegQueryCmd queries a registry key value.
func RegQueryCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	hive, path := FormatRegPath(path)
	key := cmd.Flags().Arg(1)
	session := con.GetInteractive()
	task, err := RegQuery(con.Rpc, session, hive, path, key)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("query registry key: %s\\%s\\%s", hive, path, key))
	return nil
}

func RegQuery(rpc clientrpc.MaliceRPCClient, session *core.Session, hive, path, key string) (*clientpb.Task, error) {
	request := &implantpb.RegistryRequest{
		Type: consts.ModuleRegQuery,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: fileutils.FormatWindowPath(path),
			Key:  key,
		},
	}
	return rpc.RegQuery(session.Context(), request)
}

func RegisterRegQueryFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleRegQuery,
		RegQuery,
		"breg_queryv",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, key, value, arch string) (*clientpb.Task, error) {
			hive, path := FormatRegPath(key)
			return RegQuery(rpc, sess, hive, path, key)
		},
		output.ParseResponse,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleRegQuery,
		consts.ModuleRegQuery,
		consts.ModuleRegQuery+"(active(),\"HKEY_LOCAL_MACHINE\",\"SOFTWARE\\Example\",\"TestKey\")",
		[]string{
			"session: special session",
			"hive: registry hive",
			"path: registry path",
			"key: registry",
		},
		[]string{"task"})
}
