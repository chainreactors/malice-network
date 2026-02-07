package addon

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

func ExecuteAddonCmd(cmd *cobra.Command, con *core.Console) {
	session := con.GetInteractive()
	cmdArgs := cmd.Flags().Args()
	addonName := cmd.Name()
	execArgs := cmdArgs

	if cmd.Name() == consts.ModuleExecuteAddon {
		if len(cmdArgs) == 0 {
			con.Log.Errorf("addon name is required\n")
			return
		}
		addonName = cmdArgs[0]
		execArgs = cmdArgs[1:]
	}

	timeout, _ := cmd.Flags().GetUint32("timeout")
	quiet, _ := cmd.Flags().GetBool("quiet")
	arch, _ := cmd.Flags().GetString("arch")
	process, _ := cmd.Flags().GetString("process")
	if arch == "" {
		arch = session.Os.Arch
	}

	if !session.HasAddon(addonName) {
		con.Log.Errorf("addon %s not found in %s\n", addonName, session.SessionId)
		return
	}

	addon := session.GetAddon(addonName)
	var sac *implantpb.SacrificeProcess
	if slices.Contains(consts.SacrificeModules, addon.Depend) {
		sac = common.ParseSacrificeFlags(cmd)
	}

	task, err := ExecuteAddon(con.Rpc, session, addonName, execArgs, !quiet, timeout, arch, process, sac)
	session.Console(task, string(*con.App.Shell().Line()))
	if err != nil {
		con.Log.Errorf("%s\n", err)
		return
	}
}
func ExecuteAddon(rpc clientrpc.MaliceRPCClient, sess *client.Session, name string, args []string,
	output bool, timeout uint32, arch string, process string,
	sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	if process == "" {
		process = name
	}
	return rpc.ExecuteAddon(sess.Context(), &implantpb.ExecuteAddon{
		Addon: name,
		ExecuteBinary: &implantpb.ExecuteBinary{
			Name:        name,
			Args:        args,
			Sacrifice:   sac,
			Output:      output,
			Timeout:     timeout,
			Arch:        consts.MapArch(arch),
			ProcessName: process,
			Delay:       2000,
		},
	})
}
