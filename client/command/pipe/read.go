package pipe

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// PipeReadCmd reads data from a named pipe.
func PipeReadCmd(cmd *cobra.Command, con *repl.Console) error {
	named_pipe := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := PipeRead(con.Rpc, session, named_pipe)
	if err != nil {
		return err
	}

	session.Console(cmd, task, fmt.Sprintf("read data from named pipe: %s", named_pipe))
	return nil
}

func PipeRead(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.PipeRequest{
		Type: consts.ModulePipeRead,
		Pipe: &implantpb.Pipe{
			Name: fileutils.FormatWindowPath(name),
		},
	}
	return rpc.PipeRead(session.Context(), request)
}

func RegisterPipeReadFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModulePipeRead,
		PipeRead,
		"",
		nil,
		output.ParseStatus,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModulePipeRead,
		consts.ModulePipeRead,
		consts.ModulePipeRead+`(active(), "pipe_name")`,
		[]string{"session: special session", "name: name of the pipe"},
		[]string{"task"})
}
