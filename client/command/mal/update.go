package mal

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/mals/m"
	"github.com/spf13/cobra"
	"os"
)

func UpdateMalCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	malHttpConfig := parseMalHTTPConfig(cmd)

	if name != "" {
		err := updateMal(con, name, malHttpConfig)
		if err != nil {
			return err
		}
		return nil
	}
	all, _ := cmd.Flags().GetBool("all")
	if all {
		for key := range con.MalManager.GetAllExternalPlugins() {
			err := updateMal(con, key, malHttpConfig)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func updateMal(con *repl.Console, name string, malHttpConfig m.MalHTTPConfig) error {
	var plug plugin.Plugin
	tag, err := m.GithubTagParser(RepoUrl, MalLatest, malHttpConfig)
	if err != nil {
		return err
	}
	if _, exists := con.MalManager.GetExternalPlugin(name); exists {
		updated := InstallMal(RepoUrl, name, tag, os.Stdout, malHttpConfig, con)
		if updated {
			err := con.MalManager.ReloadExternalMal(name)
			if err != nil {
				return err
			}
			plug, _ = con.MalManager.GetExternalPlugin(name)
		} else {
			return nil
		}

		plug, _ = con.MalManager.GetExternalPlugin(name)
	}
	for event, fn := range plug.GetEvents() {
		con.AddEventHook(event, fn)
	}

	for _, cmd := range plug.Commands() {
		con.ImplantMenu().AddCommand(cmd.Command)
		logs.Log.Debugf("add command: %s", cmd.Command.Name())
	}

	profile, err := assets.GetProfile()
	if err != nil {
		return err
	}
	profile.AddMal(name)
	con.Log.Importantf("load mal: %s successfully\n", name)
	return nil
}
