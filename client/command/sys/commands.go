package sys

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strconv"
	"strings"
)

func Commands(con *repl.Console) []*cobra.Command {
	whoamiCmd := &cobra.Command{
		Use:   consts.ModuleWhoami,
		Short: "Print current user",
		Run: func(cmd *cobra.Command, args []string) {
			WhoamiCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleWhoami,
		},
	}

	killCmd := &cobra.Command{
		Use:   consts.ModuleKill + " [pid]",
		Short: "Kill the process by pid",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			KillCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleKill,
		},
		Example: `kill the process which pid is 1234
~~~
kill 1234
~~~`,
	}

	common.BindArgCompletions(killCmd, nil,
		carapace.ActionValues().Usage("process pid"))

	psCmd := &cobra.Command{
		Use:   consts.ModulePs,
		Short: "List processes",
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
		Run: func(cmd *cobra.Command, args []string) {
			EnvCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleEnv,
		},
	}

	setEnvCmd := &cobra.Command{
		Use:   consts.ModuleSetEnv + " [env-key] [env-value]",
		Short: "Set environment variable",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			SetEnvCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleSetEnv,
		},
		Example: `~~~
setenv key1 value1
~~~`,
	}

	common.BindArgCompletions(setEnvCmd, nil,
		carapace.ActionValues().Usage("environment variable"),
		carapace.ActionValues().Usage("value"))

	unSetEnvCmd := &cobra.Command{
		Use:   consts.ModuleUnsetEnv + " [env-key]",
		Short: "Unset environment variable",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UnsetEnvCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleUnsetEnv,
		},
		Example: `~~~
unsetenv key1
~~~
`}

	common.BindArgCompletions(unSetEnvCmd, nil,
		carapace.ActionValues().Usage("environment variable"))

	netstatCmd := &cobra.Command{
		Use:   consts.ModuleNetstat,
		Short: "List network connections",
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
		Short: "Get basic sys info",
		Run: func(cmd *cobra.Command, args []string) {
			InfoCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleInfo,
		},
	}

	bypassCmd := &cobra.Command{
		Use:   consts.ModuleBypass,
		Short: "Bypass AMSI and ETW",
		Run: func(cmd *cobra.Command, args []string) {
			BypassCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleBypass,
		},
		Example: `
~~~
bypass --amsi --etw
~~~`,
	}

	common.BindFlag(bypassCmd, func(f *pflag.FlagSet) {
		f.Bool("amsi", false, "Bypass AMSI")
		f.Bool("etw", false, "Bypass ETW")
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
		bypassCmd,
	}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleEnv,
		Env,
		"benv",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
			return Env(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			envSet := ctx.Spite.GetResponse().GetKv()
			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				{Title: "Key", Width: 20},
				{Title: "Value", Width: 70},
			}, true)
			for k, v := range envSet {
				row = table.Row{
					k,
					v,
				}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetRows(rowEntries)
			tableModel.Title = consts.ModuleEnv
			return tableModel.View(), nil
		}, nil)

	con.RegisterImplantFunc(
		consts.ModuleSetEnv,
		SetEnv,
		"bsetenv",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, envName, value string) (*clientpb.Task, error) {
			return SetEnv(rpc, sess, envName, value)
		},
		common.ParseStatus,
		nil,
	)

	con.RegisterImplantFunc(
		consts.ModuleUnsetEnv,
		UnSetEnv,
		"bunsetenv",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, envName string) (*clientpb.Task, error) {
			return UnSetEnv(rpc, sess, envName)
		},
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleInfo,
		Info,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return ctx.Spite.GetBody(), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			info := content.Spite.GetSysinfo()
			var s strings.Builder
			s.WriteString("System Info:\n")
			s.WriteString(fmt.Sprintf("file: %s workdir: %s\n", info.Filepath, info.Workdir))
			s.WriteString(fmt.Sprintf("os: %s arch: %s, hostname: %s, username: %s\n", info.Os.Name, info.Os.Arch, info.Os.Hostname, info.Os.Username))
			s.WriteString(fmt.Sprintf("process: %s, pid: %d, ppid %d, args: %s\n", info.Process.Name, info.Process.Pid, info.Process.Ppid, info.Process.Args))
			return s.String(), nil
		})

	con.RegisterImplantFunc(
		consts.ModuleKill,
		Kill,
		"bkill",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, pid string) (*clientpb.Task, error) {
			return Kill(rpc, sess, pid)
		},
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleNetstat,
		Netstat,
		"bnetstat",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
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
		},
		func(content *clientpb.TaskContext) (string, error) {
			resp := content.Spite.GetNetstatResponse()
			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				{Title: "LocalAddr", Width: 30},
				{Title: "RemoteAddr", Width: 30},
				{Title: "SkState", Width: 20},
				{Title: "Pid", Width: 7},
				{Title: "Protocol", Width: 10},
			}, true)
			for _, sock := range resp.GetSocks() {
				row = table.Row{
					sock.LocalAddr,
					sock.RemoteAddr,
					sock.SkState,
					sock.Pid,
					sock.Protocol,
				}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.RegisterImplantFunc(
		consts.ModulePs,
		Ps,
		"bps",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
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
		},
		func(content *clientpb.TaskContext) (string, error) {
			resp := content.Spite.GetPsResponse()
			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				{Title: "Name", Width: 20},
				{Title: "PID", Width: 5},
				{Title: "PPID", Width: 5},
				{Title: "Arch", Width: 7},
				{Title: "Owner", Width: 7},
				{Title: "Path", Width: 50},
				{Title: "Args", Width: 50},
			}, true)
			for _, process := range resp.GetProcesses() {
				row = table.Row{
					process.Name,
					strconv.Itoa(int(process.Pid)),
					strconv.Itoa(int(process.Ppid)),
					process.Arch,
					process.Owner,
					process.Path,
					process.Args,
				}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.RegisterImplantFunc(
		consts.ModuleWhoami,
		Whoami,
		"bwhoami",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
			return Whoami(rpc, sess)
		},
		common.ParseResponse,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleBypass,
		Bypass,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
}
