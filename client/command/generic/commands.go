package generic

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
)

func Commands(con *repl.Console) []*cobra.Command {
	loginCmd := &cobra.Command{
		Use:   consts.CommandLogin,
		Short: "Login to server",
		Long:  help.GetHelpFor(consts.CommandLogin),
		Run: func(cmd *cobra.Command, args []string) {
			err := LoginCmd(cmd, con)
			if err != nil {
				con.App.Printf("Error login server: %s", err)
			}
		},
	}

	versionCmd := &cobra.Command{
		Use:   consts.CommandVersion,
		Short: "show server version",
		Long:  help.GetHelpFor("version"),
		Run: func(cmd *cobra.Command, args []string) {
			VersionCmd(cmd, con)
			return
		},
	}

	exitCmd := &cobra.Command{
		Use:   consts.CommandExit,
		Short: "exit client",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Exiting...")
			os.Exit(0)
			return
		},
	}

	broadcastCmd := &cobra.Command{
		Use:   consts.CommandBroadcast,
		Short: "Broadcast a message to all clients",
		Long:  help.GetHelpFor(consts.CommandBroadcast),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			BroadcastCmd(cmd, con)
		},
	}

	common.BindFlag(broadcastCmd, func(f *pflag.FlagSet) {
		f.BoolP("notify", "n", false, "notify the message to third-party services")
	})

	con.RegisterServerFunc(consts.CommandBroadcast, func(con *repl.Console, msg string) (bool, error) {
		return Broadcast(con, msg)
	})

	con.RegisterServerFunc(consts.CommandNotify, func(con *repl.Console, msg string) (bool, error) {
		return Notify(con, msg)
	})

	return []*cobra.Command{loginCmd, versionCmd, exitCmd, broadcastCmd}
}
