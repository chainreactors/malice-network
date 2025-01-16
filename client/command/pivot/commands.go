package pivot

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ReverseCmd(cmd, con)
		},
		Aliases: []string{consts.CommandSocks5},
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

	return []*cobra.Command{
		forwardCmd,
		reverseCmd,
		proxyCmd,
		rportforwardCmd,
	}
}

func Register(con *repl.Console) {
	// Register all command functions
	con.RegisterImplantFunc(
		consts.ModuleRem,
		RemDial,
		"",
		nil,
		common.ParseStatus,
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
}
