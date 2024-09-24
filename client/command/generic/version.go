package generic

import (
	"context"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func VersionCmd(cmd *cobra.Command, con *repl.Console) {
	printVersion(con)
}

func printVersion(con *repl.Console) {
	basic, err := con.Rpc.GetBasic(context.Background(), &clientpb.Empty{})
	if err != nil {
		con.Log.Errorf("Error getting version info: %v", err)
		return
	}
	con.Log.Importantf("%d.%d.%d on %s %s\n", basic.Major, basic.Minor, basic.Patch, basic.Os, basic.Arch)
}
