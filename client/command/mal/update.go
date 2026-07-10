package mal

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/mals/m"
	"github.com/spf13/cobra"
	"os"
)

func UpdateMalCmd(cmd *cobra.Command, con *core.Console) error {
	if _, err := ensureMalManager(con); err != nil {
		return err
	}

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

func updateMal(con *core.Console, name string, malHttpConfig m.MalHTTPConfig) error {
	manager, err := ensureMalManager(con)
	if err != nil {
		return err
	}
	plug, exists := manager.GetExternalPlugin(name)
	if !exists {
		return fmt.Errorf("mal %s is not loaded", name)
	}
	tag, err := m.GithubTagParser(RepoUrl, MalLatest, malHttpConfig)
	if err != nil {
		return err
	}
	updated, err := InstallMal(RepoUrl, name, tag, os.Stdout, malHttpConfig, con)
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	unregisterMalPlugin(con, con.ImplantMenu(), plug)
	err = manager.ReloadExternalMal(name)
	if err != nil {
		return err
	}
	plug, exists = manager.GetExternalPlugin(name)
	if !exists {
		return fmt.Errorf("mal %s reload completed but plugin is unavailable", name)
	}
	if err := registerMalPlugin(con, con.ImplantMenu(), plug); err != nil {
		return err
	}

	profile, err := assets.GetProfile()
	if err != nil {
		return err
	}
	profile.AddMal(name)
	con.Log.Importantf("load mal: %s successfully\n", name)
	return nil
}
