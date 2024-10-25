package basic

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func SuicideCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := Suicide(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("Suicide error: %v", err)
	}
	session.Console(task, fmt.Sprintf("%s suicide", session.SessionId))
}

func Suicide(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	return rpc.Suicide(session.Context(), &implantpb.Request{Name: consts.ModuleSuicide})
}
