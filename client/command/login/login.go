package login

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"path/filepath"
)

func Command(con *repl.Console) []*cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login to server",
		Long:  help.GetHelpFor("login"),
		Run: func(cmd *cobra.Command, args []string) {
			err := LoginCmd(cmd, con)
			if err != nil {
				con.App.Printf("Error login server: %s", err)
			}
		},
	}
	return []*cobra.Command{
		loginCmd,
	}
}

func LoginCmd(cmd *cobra.Command, con *repl.Console) error {
	files, err := assets.GetConfigs()
	if err != nil {
		con.App.Printf("Error retrieving YAML files: %s", err)
		return err
	}

	// Create a model for the interactive list
	m := tui.NewSelect(files)
	m.Title = "Select User: "
	newLogin := tui.NewModel(m, nil, false, false)
	err = newLogin.Run()
	if err != nil {
		con.App.Printf("Error running interactive list: %s", err)
		return err
	}

	// After the interactive list is completed, check the selected item
	if m.SelectedItem >= 0 && m.SelectedItem < len(m.Choices) {
		err := loginServer(con, m.Choices[m.SelectedItem])
		if err != nil {
			fmt.Println("Error executing loginServer:", err)
		}
	}

	return nil
}

func loginServer(con *repl.Console, selectedFile string) error {
	configFile := filepath.Join(assets.GetConfigDir(), selectedFile)
	config, err := mtls.ReadConfig(configFile)
	if err != nil {
		con.App.Printf("Error reading config file: %s", err)
		return err
	}

	err = con.Login(config)
	if err != nil {
		con.App.Printf("Error login: %s", err)
		return err
	}
	req := &clientpb.LoginReq{
		Name: config.Operator,
		Host: config.LHost,
		Port: uint32(config.LPort),
	}
	res, err := con.Rpc.LoginClient(context.Background(), req)
	if err != nil {
		con.App.Printf("Error login server: %s", err)
		return err
	}
	if res.Success != true {
		con.App.Printf("Error login server")
		return err
	}
	return nil
}
