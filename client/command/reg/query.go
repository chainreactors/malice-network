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
	"strings"
)

// RegQueryCmd queries a registry key value.
func RegQueryCmd(cmd *cobra.Command, con *repl.Console) error {
	hive, path, key := common.ParseRegistryFlags(cmd)

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
			Path: path,
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
			s := strings.Split(key, "\\")
			hive := s[0]
			path := strings.Join(s[1:], "\\")
			return RegQuery(rpc, sess, hive, path, key)
		},
		common.ParseResponse,
		nil,
	)
}
