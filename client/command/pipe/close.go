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

// PipeCloseCmd closes a named pipe.
func PipeCloseCmd(cmd *cobra.Command, con *repl.Console) error {
	named_pipe := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := PipeClose(con.Rpc, session, named_pipe)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("closed named pipe: %s", named_pipe))
	return nil
}

func PipeClose(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.PipeRequest{
		Type: consts.ModulePipeClose,
		Pipe: &implantpb.Pipe{
			Name: fileutils.FormatWindowPath(name),
		},
	}
	return rpc.PipeClose(session.Context(), request)
}

func RegisterPipeCloseFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModulePipeClose,
		PipeClose,
		"",
		nil,
		output.ParseStatus,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModulePipeClose,
		consts.ModulePipeClose,
		consts.ModulePipeClose+`(active(), "pipe_name")`,
		[]string{"session: special session", "name: name of the pipe"},
		[]string{"task"})
}
