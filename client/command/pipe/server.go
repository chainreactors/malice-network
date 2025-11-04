package pipe

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/session"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// PipeServerCmd manages pipe server operations (start, stop, list).
func PipeServerCmd(cmd *cobra.Command, con *repl.Console) error {
	action := cmd.Flags().Arg(0)
	var pipeName string
	if cmd.Flags().NArg() > 1 {
		pipeName = cmd.Flags().Arg(1)
	}

	task, err := PipeServer(con.Rpc, con.GetInteractive(), action, pipeName)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func PipeServer(rpc clientrpc.MaliceRPCClient, session *session.Session, action string, pipeName string) (*clientpb.Task, error) {
	// Validate action
	validActions := []string{"start", "stop", "list", "clear", "status"}
	isValid := false
	for _, validAction := range validActions {
		if action == validAction {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, fmt.Errorf("invalid action: %s. Must be one of: start, stop, list, clear, status", action)
	}

	// Validate pipe name for actions that require it
	if (action == "start" || action == "stop" || action == "clear" || action == "status") && pipeName == "" {
		return nil, fmt.Errorf("pipe name is required for %s action", action)
	}

	// Format pipe name if provided
	if pipeName != "" {
		pipeName = fileutils.FormatWindowPath(pipeName)
	}

	task, err := rpc.PipeServer(session.Context(), &implantpb.PipeRequest{
		Type: consts.ModulePipeServer,
		Pipe: &implantpb.Pipe{
			Name:   pipeName,
			Target: action, // Use target field to specify the action
		},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

// RegisterPipeServerFunc registers the pipe server function with the console
func RegisterPipeServerFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModulePipeServer,
		PipeServer,
		"",
		nil,
		output.ParseResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModulePipeServer,
		consts.ModulePipeServer,
		consts.ModulePipeServer+`(active(), "action", "pipe_name")`,
		[]string{
			"session: special session",
			"action: pipe server action (start/stop/list/clear/status)",
			"pipe_name: name of the pipe (required for start/stop/clear/status)",
		},
		[]string{"task"})
}
