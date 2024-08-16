package version

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

func Command(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name:     "version",
			Help:     "show server version",
			LongHelp: help.GetHelpFor("version"),
			Run: func(ctx *grumble.Context) error {
				VersionCmd(ctx, con)
				return nil
			},
		},
	}
}

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
