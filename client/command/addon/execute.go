package addon

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"slices"
)

func ExecuteAddonCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	name := cmd.Flags().Arg(0)
	args := cmd.Flags().Args()

	if !session.HasAddon(name) {
		repl.Log.Errorf("addon %s not found in %s", name, session.SessionId)
		return
	}

	addon := session.GetAddon(name)
	var sac *implantpb.SacrificeProcess
	if slices.Contains(consts.SacrificeModules, addon.Depend) {
		sac, _ = common.ParseSacrifice(cmd)
	}

	task, err := ExecuteAddon(con.Rpc, session, name, sac, args)
	if err != nil {
		repl.Log.Errorf("%s", err)
		return
	}

	con.AddCallback(task, func(msg proto.Message) {
		resp, _ := builtin.ParseAssembly(msg.(*implantpb.Spite))
		session.Log.Console(resp)
	})
}

func ExecuteAddon(rpc clientrpc.MaliceRPCClient, sess *repl.Session, name string, sac *implantpb.SacrificeProcess, args []string) (*clientpb.Task, error) {
	if !sess.HasAddon(name) {
		return nil, fmt.Errorf("addon %s not found in %s", name, sess.SessionId)
	}
	return rpc.ExecuteAddon(repl.Context(sess), &implantpb.ExecuteAddon{
		Addon: name,
		ExecuteBinary: &implantpb.ExecuteBinary{
			Name:      name,
			Params:    args,
			Sacrifice: sac,
			Output:    true,
		},
	})
}
