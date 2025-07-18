package sys

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"strings"
)

func InfoCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := Info(con.Rpc, session)
	if err != nil {
		return err
	}
	session.Console(cmd, task, "sysinfo")
	return nil
}

func Info(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Info(session.Context(), &implantpb.Request{
		Name: consts.ModuleSysInfo,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterInfoFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleSysInfo,
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

	con.AddCommandFuncHelper(
		consts.ModuleSysInfo,
		consts.ModuleSysInfo,
		"sysinfo(active)",
		[]string{
			"sess: special session",
		},
		[]string{"task"})
}
