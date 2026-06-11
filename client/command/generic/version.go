package generic

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func VersionCmd(cmd *cobra.Command, con *core.Console) error {
	return printVersion(con)
}

func printVersion(con *core.Console) error {
	basic, err := con.Rpc.GetBasic(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	con.Log.Importantf("%s on %s %s\n", basic.Version, basic.Os, basic.Arch)
	return nil
}
