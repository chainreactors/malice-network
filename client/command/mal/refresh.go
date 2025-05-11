package mal

import (
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func RefreshMalCmd(cmd *cobra.Command, con *repl.Console) error {
	implantCmd := con.ImplantMenu()
	for _, c := range implantCmd.Commands() {
		if c.GroupID == consts.MalGroup {
			implantCmd.RemoveCommand(c)
		}
	}

	for _, plug := range loadedMals {
		err := plug.Plugin.Destroy()
		if err != nil {
			con.Log.Warnf("Failed to destroy plugin: %s\n", err)
		}
	}

	for _, malName := range plugin.GetPluginManifest() {
		_, err := LoadMalWithManifest(con, implantCmd, malName)
		if err != nil {
			con.Log.Errorf("Failed to load mal: %s\n", err)
			continue
		}
	}
	return nil
}
