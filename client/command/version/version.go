package version

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/commonpb"
	"github.com/desertbit/grumble"
)

func VersionCmd(ctx *grumble.Context, con *console.Console) {
	printVersion(con)
}

func printVersion(con *console.Console) {
	basic, err := con.Rpc.GetBasicInfo(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Println("Error getting version info:", err)
		return
	}
	fmt.Printf("%d.%d.%d on %s %s\n", basic.Major, basic.Minor, basic.Patch, basic.OS, basic.Arch)
}
