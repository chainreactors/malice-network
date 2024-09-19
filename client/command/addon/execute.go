package addon

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"slices"
)

func ExecuteAddonCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	addonName := cmd.Flags().Arg(0)
	args := cmd.Flags().Args()[1:]
	timeout, _ := cmd.Flags().GetUint32("timeout")
	quiet, _ := cmd.Flags().GetBool("quiet")
	arch, _ := cmd.Flags().GetString("arch")
	process, _ := cmd.Flags().GetString("process")
	if !session.HasAddon(cmd.Name()) {
		con.Log.Errorf("addon %s not found in %s", addonName, session.SessionId)
		return
	}

	addon := session.GetAddon(addonName)
	var sac *implantpb.SacrificeProcess
	if slices.Contains(consts.SacrificeModules, addon.Depend) {
		sac, _ = common.ParseSacrifice(cmd)
	}

	_, err := ExecuteAddon(con.Rpc, session, addonName, args, !quiet, timeout, arch, process, sac)
	if err != nil {
		con.Log.Errorf("%s", err)
		return
	}
}
func ExecuteAddon(rpc clientrpc.MaliceRPCClient, sess *core.Session, name string, args []string,
	output bool, timeout uint32, arch string, process string,
	sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	return rpc.ExecuteAddon(sess.Context(), &implantpb.ExecuteAddon{
		Addon: name,
		ExecuteBinary: &implantpb.ExecuteBinary{
			Name:        name,
			Args:        args,
			Sacrifice:   sac,
			Output:      output,
			Timeout:     timeout,
			Arch:        consts.ArchMap[arch],
			ProcessName: process,
		},
	})
}
