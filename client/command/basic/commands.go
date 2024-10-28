package basic

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	sleepCmd := &cobra.Command{
		Use:   consts.ModuleSleep + "[interval/second]",
		Short: "change implant sleep config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return SleepCmd(cmd, con)
		},
	}

	common.BindFlag(sleepCmd, func(f *pflag.FlagSet) {
		f.Float64("jitter", 0, "jitter")
	})

	suicideCmd := &cobra.Command{
		Use:   consts.ModuleSuicide,
		Short: "kill implant",
		RunE: func(cmd *cobra.Command, args []string) error {
			return SuicideCmd(cmd, con)
		},
	}

	return []*cobra.Command{sleepCmd, suicideCmd}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(consts.ModuleSleep,
		Sleep,
		"",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, interval uint64) (*clientpb.Task, error) {
			return Sleep(rpc, sess, interval, sess.Timer.Jitter)
		},
		common.ParseStatus,
		nil,
	)

	con.RegisterImplantFunc(consts.ModuleSuicide,
		Suicide,
		"bexit",
		nil,
		common.ParseStatus,
		nil,
	)
}
