package generic

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/kballard/go-shellquote"
	"google.golang.org/protobuf/proto"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/helper/intermediate"
)

func Commands(con *core.Console) []*cobra.Command {
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return VersionCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return BroadcastCmd(cmd, con)
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

			con.Log.Console(string(out))
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

	common.BindFlag(pivotCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "list all pivot agents")
	})

	licenseInfoCmd := &cobra.Command{
		Use:   consts.CommandLicense,
		Short: "show server license info",
		Long:  "show server license info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetLicenseCmd(cmd, con)
		},
		Example: `~~~
license
~~~`,
	}

	return []*cobra.Command{loginCmd, versionCmd, exitCmd, broadcastCmd, cmdCmd, pivotCmd, licenseInfoCmd, StatusCommand(con)}
}

func Log(con *core.Console, sess *client.Session, msg string, notify bool) (bool, error) {
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

// ExecuteModule executes a dynamically constructed module request via the ExecuteModule RPC.
func ExecuteModule(rpc clientrpc.MaliceRPCClient, sess *client.Session, spite *implantpb.Spite, expect string) (*clientpb.Task, error) {
	if spite == nil {
		return nil, errors.New("spite required")
	}
	return rpc.ExecuteModule(sess.Context(), &implantpb.ExecuteModuleRequest{
		Spite:  spite,
		Expect: expect,
	})
}

func Register(con *core.Console) {
	con.RegisterServerFunc("console", func(con *core.Console) *core.Console {
		return con
	}, nil)

	con.RegisterServerFunc("sessions", func(con *core.Console) map[string]*client.Session {
		return con.Sessions
	}, nil)

	con.RegisterServerFunc("listeners", func(con *core.Console) map[string]*clientpb.Listener {
		return con.Listeners
	}, nil)

	con.RegisterServerFunc("pipelines", func(con *core.Console) map[string]*clientpb.Pipeline {
		return con.Pipelines
	}, nil)

	con.RegisterServerFunc("run", core.RunCommand, nil)

	con.RegisterServerFunc("async_run", func(con *core.Console, cmdline interface{}) (bool, error) {
		var args []string
		var err error
		switch c := cmdline.(type) {
		case string:
			args, err = shellquote.Split(c)
			if err != nil {
				return false, err
			}
		case []string:
			args = c
		}

		err = con.App.Execute(con.Context(), con.App.ActiveMenu(), args, true)
		if err != nil {
			return false, err
		}
		return true, nil
	}, nil)

	con.RegisterServerFunc(consts.CommandBroadcast, func(con *core.Console, msg string) (bool, error) {
		return Broadcast(con, &clientpb.Event{
			Type:    consts.EventBroadcast,
			Client:  con.Client,
			Message: []byte(msg),
		})
	}, nil)

	con.RegisterServerFunc(consts.CommandNotify, func(con *core.Console, msg string) (bool, error) {
		return Notify(con, &clientpb.Event{
			Type:    consts.EventNotify,
			Client:  con.Client,
			Message: []byte(msg),
		})
	}, nil)

	con.RegisterServerFunc("callback_log", func(con *core.Console, sess *client.Session, notify bool) intermediate.BuiltinCallback {
		return func(content interface{}) (interface{}, error) {
			return Log(con, sess, fmt.Sprintf("%v", content), notify)
		}
	}, nil)

	con.RegisterServerFunc("log", func(con *core.Console, sess *client.Session, msg string, notify bool) (bool, error) {
		return Log(con, sess, msg, notify)
	}, nil)

	con.RegisterServerFunc("blog", func(con *core.Console, sess *client.Session, msg string) (bool, error) {
		return Log(con, sess, msg, false)
	}, nil)

	// ExecuteModule - execute a dynamically constructed module request
	con.RegisterImplantFunc(
		"execute_module",
		ExecuteModule,
		"",
		nil,
		nil,
		nil)

	con.AddCommandFuncHelper(
		"execute_module",
		"execute_module",
		"execute_module(active(), spite, \"expect_type\")",
		[]string{
			"session: special session",
			"spite: the spite request to execute",
			"expect: expected response type name",
		},
		[]string{"task"})

	// spite - build a Spite from a proto message body
	con.RegisterServerFunc("spite", func(con *core.Console, body proto.Message) (*implantpb.Spite, error) {
		return types.BuildSpite(&implantpb.Spite{}, body)
	}, nil)

}
