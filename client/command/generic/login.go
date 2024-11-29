package generic

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
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
		return fmt.Errorf("error retrieving YAML files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no auth config found, maybe use `iom login [authfile.auth]` auto import")
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
		return Login(con, m.Choices[m.SelectedItem])
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
