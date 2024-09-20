package generic

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"os/exec"
)

func Commands(con *repl.Console) []*cobra.Command {
	loginCmd := &cobra.Command{
		Use:   consts.CommandLogin,
		Short: "Login to server",
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
		Use:   consts.CommandBroadcast + " [message]",
		Short: "Broadcast a message to all clients",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			BroadcastCmd(cmd, con)
		},
	}

	common.BindFlag(broadcastCmd, func(f *pflag.FlagSet) {
		f.BoolP("notify", "n", false, "notify the message to third-party services")
	})

	cmdCmd := &cobra.Command{
		Use:   "! [command]",
		Short: "Run a command",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// os exec

			out, err := exec.Command(args[0], args[1:]...).Output()
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			// 打印标准输出
			fmt.Println(string(out))
		},
	}

	con.RegisterServerFunc(consts.CommandBroadcast, func(con *repl.Console, msg string) (bool, error) {
		return Broadcast(con, &clientpb.Event{
			Type:    consts.EventBroadcast,
			Client:  con.Client,
			Message: msg,
		})
	})

	con.RegisterServerFunc(consts.CommandNotify, func(con *repl.Console, msg string) (bool, error) {
		return Notify(con, &clientpb.Event{
			Type:    consts.EventNotify,
			Client:  con.Client,
			Message: msg,
		})
	})

	return []*cobra.Command{loginCmd, versionCmd, exitCmd, broadcastCmd, cmdCmd}
}
