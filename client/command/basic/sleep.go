package basic

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"strconv"
)

func SleepCmd(cmd *cobra.Command, con *repl.Console) {
	interval, err := strconv.Atoi(cmd.Flags().Arg(0))
	if err != nil {
		con.Log.Errorf("Cat error: %v", err)
		return
	}
	session := con.GetInteractive()
	jitter, _ := cmd.Flags().GetFloat64("jitter")
	if jitter == 0 {
		jitter = session.Timer.Jitter
	}

	task, err := Sleep(con.Rpc, session, uint64(interval), jitter)
	if err != nil {
		con.Log.Errorf("Sleep error: %v", err)
	}

	session.Console(task, fmt.Sprintf("change sleep %d %f", interval, jitter))
}

func Sleep(rpc clientrpc.MaliceRPCClient, session *core.Session, interval uint64, jitter float64) (*clientpb.Task, error) {
	return rpc.Sleep(session.Context(), &implantpb.Timer{
		Interval: interval,
		Jitter:   jitter,
	})
}
