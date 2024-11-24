package generic

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"path/filepath"
)

func LoginCmd(cmd *cobra.Command, con *repl.Console) error {
	var err error
	if filename := cmd.Flags().Arg(0); filename != "" {
		return Login(con, filename)
	} else if filename, _ := cmd.Flags().GetString("auth"); filename != "" {
		return Login(con, filename)
	}
	files, err := assets.GetConfigs()
	if err != nil {
		con.Log.Errorf("Error retrieving YAML files: %s\n", err)
		return err
	}

	if len(files) == 0 {
		con.Log.Error("No auth config found, maybe use `iom [authfile.auth]` auto import\n")
		return nil
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
		return Login(con, configFile)
	}

	return nil
}

func Login(con *repl.Console, authFile string) error {
	config, err := assets.LoadConfig(authFile)
	if err != nil {
		return err
	}
	err = repl.Login(con, config)
	if err != nil {
		return err
	}
	return nil
}
