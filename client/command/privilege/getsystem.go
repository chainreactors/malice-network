package privilege

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

// GetSystemCmd attempts to elevate privileges.
func GetSystemCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := GetSystem(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, "attempt to elevate privileges")
	return nil
}

func GetSystem(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	request := &implantpb.Request{
		Name: consts.ModuleGetSystem,
	}
	return rpc.GetSystem(session.Context(), request)
}

func RegisterGetSystemFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleGetSystem,
		GetSystem,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
	con.AddInternalFuncHelper(
		consts.ModuleGetSystem,
		consts.ModuleGetSystem,
		consts.ModuleGetSystem+"(active())",
		[]string{
			"session: special session",
		},
		[]string{"task"})
}
