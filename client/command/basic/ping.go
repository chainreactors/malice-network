package basic

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/spf13/cobra"
)

func PingCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	_, err := Ping(con, session)
	if err != nil {
		return err
	}
	return nil
}

func Ping(con *repl.Console, sess *core.Session) (*clientpb.Task, error) {
	return con.Rpc.Ping(sess.Context(), &implantpb.Ping{Nonce: int32(cryptography.RandomInRange(0, 0x0fffffff))})
}
