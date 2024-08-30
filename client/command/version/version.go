package version

import (
	"context"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func Command(con *console.Console) []*cobra.Command {
	return []*cobra.Command{
		&cobra.Command{
			Use:   "version",
			Short: "show server version",
			Long:  help.GetHelpFor("version"),
			Run: func(cmd *cobra.Command, args []string) {
				VersionCmd(cmd, con)
				return
			},
		},
	}
}

func VersionCmd(cmd *cobra.Command, con *console.Console) {
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
