package exec

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"strings"
)

func RunCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	cmdStr := shellquote.Join(cmd.Flags().Args()...)
	task, err := Execute(con.Rpc, session, cmdStr, false, true)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, "run: "+cmdStr)
	return nil
}

func ExecuteCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	cmdStr := shellquote.Join(cmd.Flags().Args()...)
	task, err := Execute(con.Rpc, session, cmdStr, false, false)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, "execute: "+cmdStr)
	return nil
}

func Execute(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string, realtime, output bool) (*clientpb.Task, error) {
	cmdStrList, err := shellquote.Split(cmd)
	if err != nil {
		return nil, err
	}
	task, err := rpc.Execute(sess.Context(), &implantpb.ExecRequest{
		Path:     cmdStrList[0],
		Args:     cmdStrList[1:],
		Output:   output,
		Realtime: realtime,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func ShellCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	//token := ctx.Flags.Bool("token")
	quiet, _ := cmd.Flags().GetBool("quiet")
	cmdStr := strings.Join(cmd.Flags().Args(), " ")
	task, err := Shell(con.Rpc, session, cmdStr, !quiet)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, "shell: "+cmdStr)
	return nil
}

func Shell(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string, output bool) (*clientpb.Task, error) {
	var binpath string
	if sess.Os.Name == "windows" {
		binpath = `C:\Windows\System32\cmd.exe`
	} else {
		binpath = "/bin/sh"
	}

	task, err := rpc.Execute(sess.Context(), &implantpb.ExecRequest{
		Path:     binpath,
		Args:     []string{"/c", cmd},
		Output:   output,
		Realtime: true,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterExecuteFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleExecute,
		Execute,
		"",
		nil,
		output.ParseExecResponse,
		nil,
	)
	intermediate.RegisterInternalDoneCallback(consts.ModuleExecute, func(ctx *clientpb.TaskContext) (string, error) {
		resp := ctx.Spite.GetExecResponse()
		if resp.End {
			return "", nil
		}
		var s strings.Builder
		if ctx.Task.Cur == 1 {
			s.WriteString(fmt.Sprintf("pid: %d ,task: %d \n", resp.Pid, ctx.Task.TaskId))
		}
		out, err := output.ParseExecResponse(ctx)
		if err != nil {
			return "", err
		}
		s.WriteString(out.(string))
		return strings.TrimSpace(s.String()), nil
	})

	con.AddCommandFuncHelper(
		consts.ModuleExecute,
		consts.ModuleExecute,
		consts.ModuleExecute+"(active(),`whoami`,true)",
		[]string{
			"sessions",
			"cmd",
			"realtime",
			"output",
		},
		[]string{"task"},
	)

	con.RegisterAggressiveFunc(
		consts.ModuleAliasRun,
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string) (*clientpb.Task, error) {
			return Execute(con.Rpc, sess, cmd, false, true)
		},
		output.ParseExecResponse,
		nil,
	)

	con.RegisterAggressiveFunc(
		consts.ModuleAliasExecute,
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string) (*clientpb.Task, error) {
			return Execute(con.Rpc, sess, cmd, false, false)
		},
		output.ParseExecResponse,
		nil,
	)

	con.RegisterImplantFunc(
		consts.ModuleAliasShell,
		Shell,
		"bshell",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string) (*clientpb.Task, error) {
			return Shell(rpc, sess, cmd, true)
		},
		output.ParseExecResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleAliasShell,
		consts.ModuleAliasShell,
		consts.ModuleAliasShell+`(active(),"whoami",true)`,
		[]string{
			"sessions",
			"cmd",
			"output",
		}, []string{"task"})

	con.AddCommandFuncHelper(
		"bshell",
		"bshell",
		`bshell(active(),"whoami",true)`,
		[]string{
			"sessions",
			"cmd",
		},
		[]string{"task"})
}
