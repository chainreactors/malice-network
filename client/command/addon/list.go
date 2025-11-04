package addon

import (
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/session"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func AddonListCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := ListAddon(con.Rpc, session)
	session.Console(task, string(*con.App.Shell().Line()))
	if err != nil {
		con.Log.Errorf("%s\n", err)
		return
	}
}

func ListAddon(rpc clientrpc.MaliceRPCClient, sess *session.Session) (*clientpb.Task, error) {
	return rpc.ListAddon(sess.Context(), &implantpb.Request{
		Name: consts.ModuleListAddon,
	})
}
