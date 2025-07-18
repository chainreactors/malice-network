package privilege

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// GetSystemCmd attempts to elevate privileges.
func GetSystemCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := GetSystem(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(cmd, task, "attempt to elevate privileges")
	return nil
}

func GetSystem(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	return rpc.GetSystem(session.Context(), &implantpb.Request{
		Name: consts.ModuleGetSystem,
	})
}

func RegisterGetSystemFunc(con *repl.Console) {
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
