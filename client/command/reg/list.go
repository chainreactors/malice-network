package reg

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"strings"
)

// RegListKeyCmd lists the keys under a specific registry path.
func RegListKeyCmd(cmd *cobra.Command, con *core.Console) error {
	path := cmd.Flags().Arg(0)
	hive, path := FormatRegPath(path)
	session := con.GetInteractive()
	task, err := RegListKey(con.Rpc, session, hive, path)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func RegListKey(rpc clientrpc.MaliceRPCClient, session *client.Session, hive, path string) (*clientpb.Task, error) {
	request := &implantpb.RegistryRequest{
		Type: consts.ModuleRegListKey,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: fileutils.FormatWindowPath(path),
		},
	}
	return rpc.RegListKey(session.Context(), request)
}

func RegisterRegListFunc(con *core.Console) {
	con.RegisterImplantFunc(
		consts.ModuleRegListKey,
		RegListKey,
		"",
		nil,
		output.ParseArrayResponse,
		output.FormatArrayResponse,
	)
	con.RegisterImplantFunc(
		consts.ModuleRegListValue,
		RegListValue,
		"breq_query",
		func(rpc clientrpc.MaliceRPCClient, sess *client.Session, key, arch string) (*clientpb.Task, error) {
			hive, path := FormatRegPath(key)
			return RegListValue(rpc, sess, hive, path)
		},
		func(content *clientpb.TaskContext) (interface{}, error) {
			kv := content.Spite.GetResponse().GetKv()
			var s strings.Builder
			for k, v := range kv {
				s.WriteString(fmt.Sprintf("Value: %s | Data: %s\n", k, v))
			}
			return s.String(), nil
		},
		output.FormatKVResponse,
	)
	con.AddCommandFuncHelper(
		consts.ModuleRegListKey,
		consts.ModuleRegListKey,
		consts.ModuleRegListKey+"(active(),\"HKEY_LOCAL_MACHINE\",\"SOFTWARE\\Example\")",
		[]string{
			"session: special session",
			"hive: registry hive",
			"path: registry path",
		},
		[]string{"task"})

}

// RegListValueCmd lists the values under a specific registry path.
func RegListValueCmd(cmd *cobra.Command, con *core.Console) error {
	path := cmd.Flags().Arg(0)
	hive, path := FormatRegPath(path)
	session := con.GetInteractive()
	task, err := RegListValue(con.Rpc, session, hive, path)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func RegListValue(rpc clientrpc.MaliceRPCClient, session *client.Session, hive, path string) (*clientpb.Task, error) {
	request := &implantpb.RegistryRequest{
		Type: consts.ModuleRegListValue,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: fileutils.FormatWindowPath(path),
		},
	}
	return rpc.RegListValue(session.Context(), request)
}
