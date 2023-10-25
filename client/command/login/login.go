package login

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/desertbit/grumble"
)

func LoginCmd(ctx *grumble.Context, con *console.Console) {
	// TODO : interactive choice config
	config := &assets.ClientConfig{
		LHost: "127.0.0.1",
		LPort: 5004,
	}
	err := con.Login(config)
	if err != nil {
		return
	}
}
