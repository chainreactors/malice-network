package basic

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"strconv"
)

func SleepCmd(cmd *cobra.Command, con *repl.Console) error {
	interval, err := strconv.Atoi(cmd.Flags().Arg(0))
	session := con.GetInteractive()
	jitter, _ := cmd.Flags().GetFloat64("jitter")
	if jitter == 0 {
		jitter = session.Timer.Jitter
	}
	if interval < 1 {
		logs.Log.Warnf("minimum sleep interval is 1 second, auto set 1")
		interval = 1
	}
	task, err := Sleep(con.Rpc, session, uint64(interval), jitter)
	if err != nil {
		return err
	}

	session.Console(cmd, task, fmt.Sprintf("change sleep %d %f", interval, jitter))
	return nil
}

func Sleep(rpc clientrpc.MaliceRPCClient, session *core.Session, interval uint64, jitter float64) (*clientpb.Task, error) {
	return rpc.Sleep(session.Context(), &implantpb.Timer{
		Interval: interval,
		Jitter:   jitter,
	})
}
