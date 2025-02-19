package generic

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func VersionCmd(cmd *cobra.Command, con *repl.Console) {
	printVersion(con)
}

func printVersion(con *repl.Console) {
	basic, err := con.Rpc.GetBasic(con.Context(), &clientpb.Empty{})
	if err != nil {
		con.Log.Errorf("Error getting version info: %v\n", err)
		return
	}
	con.Log.Importantf("%s on %s %s\n", basic.Version, basic.Os, basic.Arch)
}
