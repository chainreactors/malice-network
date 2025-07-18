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

// Rev2selfCmd reverts to the original token.
func Rev2selfCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := Rev2self(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(cmd, task, "reverting to original token")
	return nil
}

func Rev2self(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Rev2Self(session.Context(), &implantpb.Request{
		Name: consts.ModuleRev2Self,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterRev2selfFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleRev2Self,
		Rev2self,
		"",
		nil,
		output.ParseStatus,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleRev2Self,
		consts.ModuleRev2Self,
		consts.ModuleRev2Self+`(active())`,
		[]string{
			"session: special session",
		},
		[]string{"task"})
}
