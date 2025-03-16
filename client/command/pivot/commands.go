package pivot

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	remCmd := &cobra.Command{
		Use:   consts.CommandRemDial + " [pipeline] [args]",
		Short: "Run rem on the implant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemDialCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRem,
		},
	}

	forwardCmd := &cobra.Command{
		Use:   consts.CommandPortForward + " [pipeline]",
		Short: "Forward local port to remote target",
		Long:  `Forward local port to remote target through the implant`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ForwardCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRem,
		},
		Example: `Forward local port to remote target:
~~~
forward pipeline1 --port 8080 --target 192.168.1.1:80
~~~`,
	}
	common.BindArgCompletions(forwardCmd, nil, common.RemPipelineCompleter(con))
	common.BindFlag(forwardCmd, func(f *pflag.FlagSet) {
		f.StringP("port", "p", "", "Local port to listen on")
		f.StringP("target", "t", "", "Remote target address (host:port)")
	})

	reverseCmd := &cobra.Command{
		Use:   consts.CommandReverse + " [pipeline]",
		Short: "Reverse port forward from remote to local",
		Long:  `Create a reverse port forward from remote target to local through the implant`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ReverseCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRem,
		},
		Example: `Create reverse port forward:
~~~
reverse pipeline1 --port 12345
~~~`,
	}
	common.BindArgCompletions(reverseCmd, nil, common.RemPipelineCompleter(con))
	common.BindFlag(reverseCmd, common.ProxyFlagSet)

	proxyCmd := &cobra.Command{
		Use:   consts.CommandProxy + " [pipeline]",
		Short: "Create a proxy through the implant",
		Long:  `Create a proxy server through the implant with optional authentication`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ProxyCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRem,
		},
		Example: `Create a proxy server:
~~~
proxy pipeline1 --port 8080
~~~`,
	}
	common.BindArgCompletions(proxyCmd, nil, common.RemPipelineCompleter(con))
	common.BindFlag(proxyCmd, common.ProxyFlagSet)

	rportforwardCmd := &cobra.Command{
		Use:   consts.CommandReversePortForward + " [pipeline]",
		Short: "Remote port forward through the implant",
		Long:  `Create a remote port forward through the implant to connect back to a local port`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ReversePortForwardCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRem,
		},
		Example: `Create remote port forward:
~~~
rportforward pipeline1 --port 8080 --remote 192.168.1.1:80
~~~`,
	}

	common.BindArgCompletions(rportforwardCmd, nil, common.RemPipelineCompleter(con))
	common.BindFlag(rportforwardCmd, func(f *pflag.FlagSet) {
		f.StringP("port", "p", "", "Local port to listen on")
		f.StringP("remote", "r", "", "implant's address to connect to (host:port)")
	})

	rportforwardLocalCmd := &cobra.Command{
		Use:   consts.CommandReversePortForwardLocal + " [pipeline] [agent]",
		Short: "Remote port forward through the implant to client",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RPortForwardLocalCmd(cmd, con)
		},
	}

	common.BindArgCompletions(rportforwardLocalCmd, nil,
		common.RemPipelineCompleter(con),
		common.RemAgentCompleter(con),
	)
	common.BindFlag(rportforwardLocalCmd, func(f *pflag.FlagSet) {
		f.StringP("port", "p", "", "Local port to listen on")
		f.StringP("remote", "r", "", "implant's internal address to connect to (host:port)")
	})
	rportforwardLocalCmd.MarkFlagRequired("remote")

	portforwardLocalCmd := &cobra.Command{
		Use:   consts.CommandPortForwardLocal + " [pipeline] [agent]",
		Short: "Forward local port to remote target",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PortForwardLocalCmd(cmd, con)
		},
	}
	common.BindArgCompletions(portforwardLocalCmd, nil,
		common.RemPipelineCompleter(con),
		common.RemAgentCompleter(con))
	common.BindFlag(portforwardLocalCmd, func(f *pflag.FlagSet) {
		f.StringP("port", "p", "", "Local port to listen on")
		f.StringP("local", "l", "", "Local address to connect to (host:port)")
	})
	return []*cobra.Command{
		remCmd,
		forwardCmd,
		reverseCmd,
		proxyCmd,
		rportforwardCmd,
		portforwardLocalCmd,
		rportforwardLocalCmd,
	}
}

func Register(con *repl.Console) {
	// Register all command functions
	con.RegisterImplantFunc(
		consts.ModuleRem,
		RemDial,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			resp, err := output.ParseResponse(content)
			if err != nil {
				return nil, err
			}
			return fmt.Sprintf("rem agent id: %s", resp), nil
		},
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleRem,
		consts.ModuleRem,
		consts.ModuleRem+`(active(),"pipeline1",{"-p","1080"})`,
		[]string{
			"session: special session",
			"pipeline: pipeline name",
			"args: command args",
		},
		[]string{"task"},
	)

	con.RegisterServerFunc("rem_link", GetRemLink, nil)

	con.RegisterServerFunc("pivots", generic.ListPivot, nil)
}
