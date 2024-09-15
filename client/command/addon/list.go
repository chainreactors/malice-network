package addon

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"strings"
)

func AddonListCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := ListAddon(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("%s", err)
		return
	}

	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		exts := msg.GetAddons()
		if len(exts.Addons) == 0 {
			return "", fmt.Errorf("No addon found.")
		}
		session.Addons = exts
		var s strings.Builder
		for _, ext := range exts.Addons {
			s.WriteString(fmt.Sprintf("%s\t%s\t%s\n", ext.Name, ext.Type, ext.Depend))
		}
		return s.String(), nil
	})
}

func ListAddon(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
	return rpc.ListAddon(sess.Context(), &implantpb.Request{
		Name: consts.ModuleListAddon,
	})
}
