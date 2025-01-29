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

// PrivsCmd lists available privileges.
func PrivsCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := Privs(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, "list available privileges")
	return nil
}

func Privs(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	request := &implantpb.Request{
		Name: consts.ModulePrivs,
	}
	return rpc.Privs(session.Context(), request)
}

func RegisterPrivsFunc(con *repl.Console) {
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
