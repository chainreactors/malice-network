package login

import (
	"context"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"path/filepath"
)

func Command(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: "login",
			Help: "Login to server",
			Flags: func(f *grumble.Flags) {
				f.String("c", "config", "", "server config")
			},
			Run: func(ctx *grumble.Context) error {
				err := LoginCmd(ctx, con)
				if err != nil {
					con.App.Println("Error login server: ", err)
				}
				return nil
			},
		},
	}
}

func LoginCmd(ctx *grumble.Context, con *console.Console) error {
	files, err := assets.GetConfigs()
	if err != nil {
		con.App.Println("Error retrieving YAML files:", err)
		return err
	}

	// Create a model for the interactive list
	m := tui.NewSelect(files)
	m.Title = "Select a user: "
	newLogin := tui.NewModel(m, nil, false, false)
	err = newLogin.Run()
	if err != nil {
		con.App.Println("Error running interactive list:", err)
		return err
	}

	// After the interactive list is completed, check the selected item
	if m.SelectedItem >= 0 && m.SelectedItem < len(m.Choices) {
		err := loginServer(ctx, con, m.Choices[m.SelectedItem])
		if err != nil {
			fmt.Println("Error executing loginServer:", err)
		}
	}

	return nil
}

func loginServer(ctx *grumble.Context, con *console.Console, selectedFile string) error {
	configFile := filepath.Join(assets.GetConfigDir(), selectedFile)
	config, err := assets.ReadConfig(configFile)
	if err != nil {
		con.App.Println("Error reading config file:", err)
		return err
	}

	err = con.Login(config)
	if err != nil {
		con.App.Println("Error login:", err)
		return err
	}
	req := &clientpb.LoginReq{
		Name: config.Operator,
		Host: config.LHost,
		Port: uint32(config.LPort),
	}
	res, err := con.Rpc.LoginClient(context.Background(), req)
	if err != nil {
		con.App.Println("Error login server: ", err)
		return err
	}
	if res.Success != true {
		con.App.Println("Error login server")
		return err
	}
	return nil
}
