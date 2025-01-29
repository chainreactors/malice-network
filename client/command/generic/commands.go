package generic

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/mals"
)

func Commands(con *repl.Console) []*cobra.Command {
	loginCmd := &cobra.Command{
		Use:   consts.CommandLogin,
		Short: "Login to server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return LoginCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// os exec

			if len(args) < 1 {
				return errors.New("command requires one or more arguments")
			}

			// Above, the length of args is checked to be at least 2
			path, err := exec.LookPath(args[0])
			if err != nil {
				return err
			}

			shellCmd := exec.Command(path, args[1:]...)

			// Load OS environment
			shellCmd.Env = os.Environ()

			out, err := shellCmd.CombinedOutput()
			if err != nil {
				return err
			}

			fmt.Print(string(out))
			return nil
		},
	}
	fileAction := carapace.ActionFiles()
	common.BindArgCompletions(cmdCmd, &fileAction)

	pivotCmd := &cobra.Command{
		Use:   consts.CommandPivot,
		Short: "List all pivot agents",
		Long:  "List all active pivot agents with their details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListPivotCmd(cmd, con)
		},
		Example: `List all pivot agents:
~~~
pivot
~~~`,
	}

	return []*cobra.Command{loginCmd, versionCmd, exitCmd, broadcastCmd, cmdCmd, pivotCmd}
}

func Log(con *repl.Console, sess *core.Session, msg string, notify bool) (bool, error) {
	_, err := con.Rpc.SessionEvent(sess.Context(), &clientpb.Event{
		Type:    consts.EventSession,
		Op:      consts.CtrlSessionLog,
		Session: sess.Session,
		Client:  con.Client,
		Message: []byte(msg),
	})
	if err != nil {
		return false, err
	}
	if notify {
		return Notify(con, &clientpb.Event{
			Type:    consts.EventNotify,
			Client:  con.Client,
			Message: []byte(msg),
		})
	}
	return true, nil
}

func Register(con *repl.Console) {
	con.RegisterServerFunc(consts.CommandBroadcast, func(con *repl.Console, msg string) (bool, error) {
		return Broadcast(con, &clientpb.Event{
			Type:    consts.EventBroadcast,
			Client:  con.Client,
			Message: []byte(msg),
		})
	}, nil)

	con.RegisterServerFunc(consts.CommandNotify, func(con *repl.Console, msg string) (bool, error) {
		return Notify(con, &clientpb.Event{
			Type:    consts.EventNotify,
			Client:  con.Client,
			Message: []byte(msg),
		})
	}, nil)

	con.RegisterServerFunc("callback_log", func(con *repl.Console, sess *core.Session, notify bool) intermediate.BuiltinCallback {
		return func(content interface{}) (bool, error) {
			return Log(con, sess, fmt.Sprintf("%v", content), notify)
		}
	}, nil)

	con.RegisterServerFunc("log", func(con *repl.Console, sess *core.Session, msg string, notify bool) (bool, error) {
		return Log(con, sess, msg, notify)
	}, nil)

	con.RegisterServerFunc("blog", func(con *repl.Console, sess *core.Session, msg string) (bool, error) {
		return Log(con, sess, msg, false)
	}, nil)

	con.RegisterServerFunc("barch", func(con *repl.Console, sess *core.Session) (string, error) {
		return sess.Os.Arch, nil
	}, nil)

	con.RegisterServerFunc("active", func(con *repl.Console) (*core.Session, error) {
		return con.GetInteractive().Clone(consts.CalleeMal), nil
	}, &mals.Helper{
		Short:   "get current session",
		Output:  []string{"sess"},
		Example: "active()",
	})

	con.RegisterServerFunc("is64", func(con *repl.Console, sess *core.Session) (bool, error) {
		return sess.Os.Arch == "x64", nil
	}, nil)

	con.RegisterServerFunc("isactive", func(con *repl.Console, sess *core.Session) (bool, error) {
		return sess.IsAlive, nil
	}, nil)

	con.RegisterServerFunc("isadmin", func(con *repl.Console, sess *core.Session) (bool, error) {
		return sess.IsPrivilege, nil
	}, nil)

	con.RegisterServerFunc("isbeacon", func(con *repl.Console, sess *core.Session) (bool, error) {
		return sess.Type == consts.CommandBuildBeacon, nil
	}, nil)

	con.RegisterServerFunc("bdata", func(con *repl.Console, sess *core.Session) (map[string]interface{}, error) {
		if sess == nil {
			return nil, errors.New("session is nil")
		}
		return sess.Data.Any, nil
	}, &mals.Helper{
		Short:   "get session custom data",
		Output:  []string{"map[string]interface{}"},
		Example: "bdata(active())",
	})
	con.RegisterServerFunc("data", func(con *repl.Console, sess *core.Session) (map[string]interface{}, error) {
		if sess == nil {
			return nil, errors.New("session is nil")
		}

		return sess.Data.Data(), nil
	}, &mals.Helper{
		Short:   "get session data",
		Output:  []string{"map[string]interface{}"},
		Example: "data(active())",
	})
}
