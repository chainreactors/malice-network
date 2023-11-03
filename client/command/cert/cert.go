package cert

import (
	"context"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/desertbit/grumble"
)

func CertCmd(ctx *grumble.Context, con *console.Console) {
	registerCA(ctx, con)
}

func registerCA(ctx *grumble.Context, con *console.Console) {
	host := ctx.Flags.String("host")
	user := ctx.Flags.String("user")
	port := ctx.Flags.Int("port")
	req := &clientpb.RegisterReq{
		Host: host,
		User: user,
	}
	res, err := con.Rpc.RegisterCA(context.Background(), req)
	if err != nil {
		con.App.Println("Error registering CA:", err)
		return
	}
	if certErr := certs.SaveToPEMFile(host+user+".crt", res.Certs); certErr != nil {
		con.App.Println("Error saving cert:", certErr)
		return
	}
	if keyErr := certs.SaveToPEMFile(host+user+".key", res.PrivateKey); keyErr != nil {
		con.App.Println("Error saving cert:", keyErr)
		return
	}
	err = assets.NewConfig(host, user, port, res.Certs, res.PrivateKey, res.CA)
	if err != nil {
		con.App.Println("Error saving config:", err)
		return
	}
}
