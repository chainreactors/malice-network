package login

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/desertbit/grumble"
)

func LoginCmd(ctx *grumble.Context, con *console.Console) {
	// TODO : interactive choice config
	config := &assets.ClientConfig{
		LHost: "127.0.0.1",
		LPort: 51004,
	}
	con.Login(config)
	core.Sessions.Update(con)
}
