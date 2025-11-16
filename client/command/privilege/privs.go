package privilege

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// PrivsCmd lists available privileges.
func PrivsCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	task, err := Privs(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Privs(rpc clientrpc.MaliceRPCClient, session *client.Session) (*clientpb.Task, error) {
	request := &implantpb.Request{
		Name: consts.ModulePrivs,
	}
	return rpc.Privs(session.Context(), request)
}

func RegisterPrivsFunc(con *core.Console) {
	con.RegisterImplantFunc(
		consts.ModulePrivs,
		Privs,
		"",
		nil,
		output.ParseKVResponse,
		output.FormatKVResponse,
	)
	con.AddCommandFuncHelper(
		consts.ModulePrivs,
		consts.ModulePrivs,
		consts.ModulePrivs+"(active())",
		[]string{
			"session: special session",
		},
		[]string{"task"})
}
