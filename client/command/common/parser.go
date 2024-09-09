package common

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

func ParseSacrifice(cmd *cobra.Command) (*implantpb.SacrificeProcess, error) {
	params := cmd.Flags().Args()[1:]
	ppid, _ := cmd.Flags().GetUint("ppid")
	processname, _ := cmd.Flags().GetString("process")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	return builtin.NewSacrificeProcessMessage(processname, int64(ppid), isBlockDll, argue, shellquote.Join(params...))
}

func ParseAssembly(ctx *clientpb.TaskContext) (interface{}, error) {
	return builtin.ParseAssembly(ctx.Spite)
}

func ParseStatus(ctx *clientpb.TaskContext) (interface{}, error) {
	return builtin.ParseStatus(ctx.Spite)
}

func ParseResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	resp := ctx.Spite.GetResponse()
	if resp != nil {
		return resp.GetOutput(), nil
	}
	return nil, fmt.Errorf("not response")
}
