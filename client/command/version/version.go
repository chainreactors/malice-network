package version

import (
	"context"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/desertbit/grumble"
)

func VersionCmd(ctx *grumble.Context, con *console.Console) {
	printVersion(con)
}

func printVersion(con *console.Console) {
	basic, err := con.Rpc.GetBasic(context.Background(), &clientpb.Empty{})
	if err != nil {
		console.Log.Errorf("Error getting version info: %v", err)
		return
	}
	console.Log.Importantf("%d.%d.%d on %s %s\n", basic.Major, basic.Minor, basic.Patch, basic.Os, basic.Arch)
}
