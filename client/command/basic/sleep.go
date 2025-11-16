package basic

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/gorhill/cronexpr"
	"github.com/spf13/cobra"
	"strconv"
)

func SleepCmd(cmd *cobra.Command, con *core.Console) error {
	expression := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	jitter, _ := cmd.Flags().GetFloat64("jitter")
	if jitter == 0 {
		jitter = session.Timer.Jitter
	}

	task, err := Sleep(con.Rpc, session, expression, jitter)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Sleep(rpc clientrpc.MaliceRPCClient, session *client.Session, expression string, jitter float64) (*clientpb.Task, error) {
	if _, err := strconv.Atoi(expression); err == nil {
		expression = fmt.Sprintf("*/%s * * * * * *", expression)
	}
	_, err := cronexpr.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("Invalid cron expression: %s\n", expression)
	}
	return rpc.Sleep(session.Context(), &implantpb.Timer{
		Expression: expression,
		Jitter:     jitter,
	})
}
