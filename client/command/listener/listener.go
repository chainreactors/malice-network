package listener

import (
	"context"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/desertbit/grumble"
)

func ListenerCmd(ctx *grumble.Context, con *console.Console) {
	listeners, err := con.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		return
	}
	printListeners(listeners)
}

func printListeners(listeners *clientpb.Listeners) {

}
