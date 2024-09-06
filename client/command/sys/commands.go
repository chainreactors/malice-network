package sys

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"strings"
)

func Commands(con *console.Console) []*cobra.Command {
	whoamiCmd := &cobra.Command{
		Use:   consts.ModuleWhoami,
		Short: "Print current user",
		Long:  help.GetHelpFor(consts.ModuleWhoami),
		Run: func(cmd *cobra.Command, args []string) {
			WhoamiCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleWhoami,
		},
	}

	killCmd := &cobra.Command{
		Use:   consts.ModuleKill,
		Short: "Kill the process",
		Long:  help.GetHelpFor(consts.ModuleKill),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			KillCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleKill,
		},
	}

	common.BindArgCompletions(killCmd, nil,
		carapace.ActionValues().Usage("process pid"))

	psCmd := &cobra.Command{
		Use:   consts.ModulePs,
		Short: "List processes",
		Long:  help.GetHelpFor(consts.ModulePs),
		Run: func(cmd *cobra.Command, args []string) {
			PsCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModulePs,
		},
	}

	envCmd := &cobra.Command{
		Use:   consts.ModuleEnv,
		Short: "List environment variables",
		Long:  help.GetHelpFor(consts.ModuleEnv),
		Run: func(cmd *cobra.Command, args []string) {
			EnvCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleEnv,
		},
	}

	setEnvCmd := &cobra.Command{
		Use:   consts.ModuleSetEnv,
		Short: "Set environment variable",
		Long:  help.GetHelpFor(consts.ModuleSetEnv),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			SetEnvCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleSetEnv,
		},
	}

	common.BindArgCompletions(setEnvCmd, nil,
		carapace.ActionValues().Usage("environment variable"),
		carapace.ActionValues().Usage("value"))

	unSetEnvCmd := &cobra.Command{
		Use:   consts.ModuleUnsetEnv,
		Short: "Unset environment variable",
		Long:  help.GetHelpFor(consts.ModuleUnsetEnv),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UnsetEnvCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleUnsetEnv,
		},
	}

	common.BindArgCompletions(unSetEnvCmd, nil,
		carapace.ActionValues().Usage("environment variable"))

	netstatCmd := &cobra.Command{
		Use:   consts.ModuleNetstat,
		Short: "List network connections",
		Long:  help.GetHelpFor(consts.ModuleNetstat),
		Run: func(cmd *cobra.Command, args []string) {
			NetstatCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleNetstat,
		},
	}

	infoCmd := &cobra.Command{
		Use:   consts.ModuleInfo,
		Short: "get basic sys info",
		Long:  help.GetHelpFor(consts.ModuleInfo),
		Run: func(cmd *cobra.Command, args []string) {
			InfoCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleInfo,
		},
	}

	con.RegisterInternalFunc(
		"env",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session) (*clientpb.Task, error) {
			return Env(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			envSet := ctx.Spite.GetResponse().GetKv()
			var envs []string
			for k, v := range envSet {
				envs = append(envs, fmt.Sprintf("%s:%s", k, v))
			}
			return strings.Join(envs, ","), nil
		})

	con.RegisterInternalFunc(
		"bsetenv",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, envName, value string) (*clientpb.Task, error) {
			return SetEnv(rpc, sess, envName, value)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		},
	)

	con.RegisterInternalFunc(
		"unsetenv",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, envName string) (*clientpb.Task, error) {
			return UnSetEnv(rpc, sess, envName)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"info",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session) (*clientpb.Task, error) {
			return Info(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return ctx.Spite.GetBody(), nil
		})

	con.RegisterInternalFunc(
		"bkill",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, pid string) (*clientpb.Task, error) {
			return Kill(rpc, sess, pid)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"netstat",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session) (*clientpb.Task, error) {
			return Netstat(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			netstatSet := ctx.Spite.GetNetstatResponse()
			var socks []string
			for _, sock := range netstatSet.GetSocks() {
				socks = append(socks, fmt.Sprintf("%s:%s:%s:%s:%s",
					sock.LocalAddr,
					sock.RemoteAddr,
					sock.SkState,
					sock.Pid,
					sock.Protocol))
			}
			return strings.Join(socks, ","), nil
		})

	con.RegisterInternalFunc(
		"bps",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session) (*clientpb.Task, error) {
			return Ps(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			psSet := ctx.Spite.GetPsResponse()
			var ps []string
			for _, p := range psSet.GetProcesses() {
				ps = append(ps, fmt.Sprintf("%s:%v:%v:%s:%s:%s:%s",
					p.Name,
					p.Pid,
					p.Ppid,
					p.Arch,
					p.Owner,
					p.Path,
					p.Args))
			}
			return strings.Join(ps, ","), nil
		})

	con.RegisterInternalFunc(
		"whoami",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session) (*clientpb.Task, error) {
			return Whoami(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return ctx.Spite.GetResponse().GetOutput(), nil
		})

	return []*cobra.Command{
		whoamiCmd,
		killCmd,
		psCmd,
		envCmd,
		setEnvCmd,
		unSetEnvCmd,
		netstatCmd,
		infoCmd,
	}
}
