package generic

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"path/filepath"
)

func LoginCmd(cmd *cobra.Command, con *repl.Console) error {
	files, err := assets.GetConfigs()
	if err != nil {
		con.Log.Errorf("Error retrieving YAML files: %s", err)
		return err
	}

	// Create a model for the interactive list
	m := tui.NewSelect(files)
	m.Title = "Select User: "
	newLogin := tui.NewModel(m, nil, false, false)
	err = newLogin.Run()
	if err != nil {
		con.Log.Errorf("Error running interactive list: %s", err)
		return err
	}

	// After the interactive list is completed, check the selected item
	if m.SelectedItem >= 0 && m.SelectedItem < len(m.Choices) {
		configFile := filepath.Join(assets.GetConfigDir(), m.Choices[m.SelectedItem])
		config, err := mtls.ReadConfig(configFile)
		if err != nil {
			con.Log.Errorf("Error reading config file: %s", err)
			return err
		}
		err = repl.Login(con, config)
		if err != nil {
			con.Log.Errorf("Error executing loginServer: %s", err)
		}
	}

	return nil
}
