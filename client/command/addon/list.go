package addon

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func AddonListCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := ListAddon(con.Rpc, session)
	if err != nil {
		repl.Log.Errorf("%s", err)
		return
	}

	con.AddCallback(task, func(msg proto.Message) {
		exts := msg.(*implantpb.Spite).GetAddons()
		if len(exts.Addons) == 0 {
			session.Log.Warn("No addon found.")
			return
		}
		session.Addons = exts
		for _, ext := range exts.Addons {
			session.Log.Consolef("%s\t%s\t%s", ext.Name, ext.Type, ext.Depend)
		}
	})
}

func ListAddon(rpc clientrpc.MaliceRPCClient, sess *repl.Session) (*clientpb.Task, error) {
	return rpc.ListAddon(sess.Context(), &implantpb.Request{
		Name: consts.ModuleListAddon,
	})
}
