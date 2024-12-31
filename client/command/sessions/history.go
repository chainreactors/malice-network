package sessions

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"strconv"
)

func historyCmd(cmd *cobra.Command, con *repl.Console) error {
	if con.GetInteractive() == nil {
		return fmt.Errorf("No session selected")
	}

	rawLen := cmd.Flags().Arg(0)
	if rawLen == "" {
		rawLen = "10"
	}
	length, err := strconv.Atoi(rawLen)
	if err != nil {
		return err
	}
	sess := con.GetInteractive()
	contexts, err := con.Rpc.GetSessionHistory(sess.Context(), &clientpb.Int{
		Limit: int32(length),
	})
	if err != nil {
		return err
	}
	for _, context := range contexts.Contexts {
		core.HandlerTask(sess, context, []byte{}, consts.CalleeCMD, true)
	}
	return nil
}
