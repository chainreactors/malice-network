package reg

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// RegDeleteCmd deletes a registry key.
func RegDeleteCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	hive, path := FormatRegPath(path)
	key := cmd.Flags().Arg(1)
	session := con.GetInteractive()
	task, err := RegDelete(con.Rpc, session, hive, path, key)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func RegDelete(rpc clientrpc.MaliceRPCClient, session *client.Session, hive, path, key string) (*clientpb.Task, error) {
	request := &implantpb.RegistryRequest{
		Type: consts.ModuleRegDelete,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: fileutils.FormatWindowPath(path),
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
		output.ParseStatus,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleRegDelete,
		consts.ModuleRegDelete,
		consts.ModuleRegDelete+"(active(),\"HKEY_LOCAL_MACHINE\",\"SOFTWARE\\Example\",\"TestKey\")",
		[]string{
			"session: special session",
			"hive: registry hive",
			"path: registry path",
			"key: registry key",
		},
		[]string{"task"})
}
