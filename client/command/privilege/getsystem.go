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

// GetSystemCmd attempts to elevate privileges.
func GetSystemCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	task, err := GetSystem(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func GetSystem(rpc clientrpc.MaliceRPCClient, session *client.Session) (*clientpb.Task, error) {
	return rpc.GetSystem(session.Context(), &implantpb.Request{
		Name: consts.ModuleGetSystem,
	})
}

func RegisterGetSystemFunc(con *core.Console) {
	con.RegisterImplantFunc(
		consts.ModuleGetSystem,
		GetSystem,
		"",
		nil,
		output.ParseResponse,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleGetSystem,
		consts.ModuleGetSystem,
		consts.ModuleGetSystem+"(active())",
		[]string{
			"session: special session",
		},
		[]string{"task"})
}
