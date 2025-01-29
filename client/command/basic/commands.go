package basic

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	sleepCmd := &cobra.Command{
		Use:   consts.ModuleSleep + " [interval/second]",
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

	getCmd := &cobra.Command{
		Use:   consts.ModulePing,
		Short: "get bind implant response",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetCmd(cmd, con)
		},
		Annotations: map[string]string{
			"implant": consts.ImplantMaleficBind,
		},
	}

	waitCmd := &cobra.Command{
		Use:   consts.CommandWait + " [task_id1] [task_id2]",
		Short: "wait for task to finish",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return WaitCmd(cmd, con)
		},
		Annotations: map[string]string{
			"implant": consts.ImplantMaleficBind,
		},
	}
	common.BindFlag(waitCmd, func(f *pflag.FlagSet) {
		f.Int("interval", 1, "interval")
	})
	taskComp := common.SessionTaskCompleter(con)
	common.BindArgCompletions(waitCmd, &taskComp)

	pollingCmd := &cobra.Command{
		Use:   consts.CommandPolling,
		Short: "polling task status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return PollingCmd(cmd, con)
		},
		Annotations: map[string]string{
			"implant": consts.ImplantMaleficBind,
		},
	}
	common.BindFlag(pollingCmd, func(f *pflag.FlagSet) {
		f.Int("interval", 1, "interval")
	})

	recoverCmd := &cobra.Command{
		Use:   consts.CommandRecover,
		Short: "recover session",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RecoverCmd(cmd, con)
		},
	}

	initCmd := &cobra.Command{
		Use:   consts.ModuleInit,
		Short: "init session",
		RunE: func(cmd *cobra.Command, args []string) error {
			return InitCmd(cmd, con)
		},
		Annotations: map[string]string{
			"implant": consts.ImplantMaleficBind,
		},
	}

	infoCommand := &cobra.Command{
		Use:   "info",
		Short: "show session info",
		Long:  "Displays the specified session info.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return SessionInfoCmd(cmd, con)
		},
	}
	return []*cobra.Command{sleepCmd, suicideCmd, getCmd, waitCmd, pollingCmd, initCmd, recoverCmd, infoCommand}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(consts.ModuleSleep,
		Sleep,
		"bsleep",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, interval uint64) (*clientpb.Task, error) {
			return Sleep(rpc, sess, interval, sess.Timer.Jitter)
		},
		output.ParseStatus,
		nil,
	)

	con.AddCommandFuncHelper(consts.ModuleSleep, consts.ModuleSleep,
		`sleep(active(), 10, 0.5)`,
		[]string{
			"sess:special session",
			"interval:time interval, in seconds",
			"jitter:jitter, percentage of interval",
		}, []string{"task"})

	con.AddCommandFuncHelper(
		"bsleep",
		"bsleep",
		`sleep(active(), 10)`,
		[]string{
			"sess:special session",
			"interval:time interval, in seconds",
		}, []string{"task"})

	con.RegisterImplantFunc(consts.ModuleSuicide,
		Suicide,
		"bexit",
		nil,
		output.ParseStatus,
		nil,
	)

	con.AddCommandFuncHelper(consts.ModuleSuicide, consts.ModuleSuicide,
		`suicide(active())`,
		[]string{
			"sess:special session",
		}, []string{"task"})

	con.AddCommandFuncHelper("bexit", "bexit",
		`bexit(active())`,
		[]string{
			"sess:special session",
		}, []string{"task"})
}
