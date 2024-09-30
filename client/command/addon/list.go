package addon

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func AddonListCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	_, err := ListAddon(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("%s", err)
		return
	}
}

func ListAddon(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
	return rpc.ListAddon(sess.Context(), &implantpb.Request{
		Name: consts.ModuleListAddon,
	})
}
