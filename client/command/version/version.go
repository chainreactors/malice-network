package version

import (
	"context"
	"fmt"
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
		fmt.Println("Error getting version info:", err)
		return
	}
	fmt.Printf("%d.%d.%d on %s %s\n", basic.Major, basic.Minor, basic.Patch, basic.Os, basic.Arch)
}
