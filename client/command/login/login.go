package login

import (
	"context"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/desertbit/grumble"
)

func LoginCmd(ctx *grumble.Context, con *console.Console) {
	// TODO : interactive choice config
	//config := &assets.ClientConfig{
	//	LHost: "127.0.0.1",
	//	LPort: 5004,
	//}
	//err := con.Login(config)
	//if err != nil {
	//	return
	//}
	loginServer(ctx, con)
}

func loginServer(ctx *grumble.Context, con *console.Console) {
	configFile := ctx.Flags.String("config")
	config, err := assets.ReadConfig(configFile)
	if err != nil {
		con.App.Println("Error reading config file:", err)
		return
	}
	rpc, ln, err := utils.MTLSConnect(config)
	req := &clientpb.LoginReq{
		//User: config.Operator,
		Host: config.LHost,
		Port: uint32(config.LPort),
	}
	res, err := rpc.AddClient(context.Background(), req)
	if err != nil {
		con.App.Println("Error login server: ", err)
		return
	}
	defer ln.Close()
	if res.Success != true {
		con.App.Println("Error login server")
		return
	}
}
