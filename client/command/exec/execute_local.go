package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"strings"
)

// ExecuteLocalCmd - Execute local PE on sacrifice process
func ExecuteLocalCmd(cmd *cobra.Command, con *repl.Console) error {
	args := cmd.Flags().Args()
	process, _ := cmd.Flags().GetString("process")
	quiet, _ := cmd.Flags().GetBool("quiet")
	sac := common.ParseSacrificeFlags(cmd)
	task, err := ExecLocal(con.Rpc, con.GetInteractive(), args, !quiet, process, sac)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, strings.Join(args, " "))
	return nil
}

func ExecLocal(rpc clientrpc.MaliceRPCClient, sess *core.Session,
	args []string, output bool, process string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	args[0] = fileutils.FormatWindowPath(args[0])
	if process == "" {
		process = args[0]
	}

	binary := &implantpb.ExecuteBinary{
		ProcessName: process,
		Bin:         []byte(args[0]),
		Args:        args[1:],
		Output:      output,
		Sacrifice:   sac,
		Type:        consts.ModuleExecuteLocal,
	}

	task, err := rpc.ExecuteLocal(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func InlineLocalCmd(cmd *cobra.Command, con *repl.Console) error {
	args := cmd.Flags().Args()
	process, _ := cmd.Flags().GetString("process")
	output, _ := cmd.Flags().GetBool("output")
	task, err := InlineLocal(con.Rpc, con.GetInteractive(), args, output, process)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, strings.Join(args, " "))
	return nil
}

func InlineLocal(rpc clientrpc.MaliceRPCClient, sess *core.Session,
	args []string, output bool, process string) (*clientpb.Task, error) {
	args[0] = fileutils.FormatWindowPath(args[0])

	binary := &implantpb.ExecuteBinary{
		ProcessName: args[0],
		Args:        args[1:],
		Output:      output,
		Type:        consts.ModuleInlineLocal,
	}

	task, err := rpc.InlineLocal(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterExecuteLocalFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleExecuteLocal,
		ExecLocal,
		"bexecute",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmdline string) (*clientpb.Task, error) {
			args, err := shellquote.Split(cmdline)
			if err != nil {
				return nil, err
			}
			return ExecLocal(rpc, sess, args, true, "", &implantpb.SacrificeProcess{
				Hidden:   false,
				BlockDll: false,
				Etw:      false,
				Ppid:     0,
				Argue:    "",
			})
		},
		common.ParseExecResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleExecuteLocal,
		consts.ModuleExecuteLocal,
		consts.ModuleExecuteLocal+`(active(),{"-i","127.0.0.1","-p","top2"},true,"gogo.exe",new_sacrifice(1234,false,true,true,"argue"))`,
		[]string{
			"session: special session",
			"args: arguments",
			"output",
			"process",
			"sacrifice: sacrifice process",
		},
		[]string{"task"})

	con.AddCommandFuncHelper(
		"bexecute",
		"bexecute",
		`bexecute(active(),"whoami")`,
		[]string{
			"session: special session",
			"cmd: command to execute",
		},
		[]string{"task"})

	// inlinelocal
	con.RegisterImplantFunc(
		consts.ModuleInlineLocal,
		InlineLocal,
		"",
		nil,
		common.ParseExecResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleInlineLocal,
		consts.ModuleInlineLocal,
		consts.ModuleInlineLocal+`(active(),{""},true,"whoami")`,
		[]string{
			"session: special session",
			"args: arguments",
			"output",
			"process",
		},
		[]string{"task"},
	)
}
