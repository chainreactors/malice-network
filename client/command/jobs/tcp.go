package jobs

import (
	"context"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/desertbit/grumble"
)

func TcpPipelineCmd(ctx *grumble.Context, con *console.Console) {
	lhost := ctx.Flags.String("lhost")
	lport := uint16(ctx.Flags.Int("lport"))

	console.Log.Info("Starting mTLS listener ...")
	_, err := con.Rpc.StartTcpPipeline(context.Background(), &lispb.TCPPipeline{
		Host: lhost,
		Port: uint32(lport),
	})

	if err != nil {
		console.Log.Error(err.Error())
	}
}
