package pipe

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

// Commands initializes and returns all pipe-related commands.
func Commands(con *repl.Console) []*cobra.Command {
	pipeCmd := &cobra.Command{
		Use:   consts.CommandPipe,
		Short: "Manage named pipes",
		Long:  "Perform operations related to named pipes, including uploading, reading, and closing pipes.",
	}

	pipeUploadCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModulePipeUpload) + " [pipe_name] [file_path]",
		Short: "Upload file to a named pipe",
		Long:  "Upload the content of a specified file to a named pipe.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PipeUploadCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModulePipeUpload,
			"ttp":    "T1090",
		},
		Example: `Upload file to pipe:
  ~~~
  pipe upload \\.\pipe\test_pipe /path/to/file
  ~~~`,
	}

	common.BindArgCompletions(pipeUploadCmd, nil,
		carapace.ActionValues().Usage("pipe name"),
		carapace.ActionFiles().Usage("local file"))

	pipeReadCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModulePipeRead) + " [pipe_name]",
		Short: "Read data from a named pipe",
		Long:  "Read data from a specified named pipe.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PipeReadCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModulePipeRead,
			"ttp":    "T1090",
		},
		Example: `Read data from pipe:
  ~~~
  pipe read \\.\pipe\test_pipe
  ~~~`,
	}
	common.BindArgCompletions(pipeReadCmd, nil,
		carapace.ActionValues().Usage("pipe name"),
	)

	pipeCloseCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModulePipeClose) + " [pipe_name]",
		Short: "Close a named pipe",
		Long:  "Close a specified named pipe.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PipeCloseCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModulePipeClose,
			"ttp":    "T1090",
		},
		Example: `Close a pipe:
	~~~
	pipe close \\.\pipe\test_pipe
	~~~`,
	}
	common.BindArgCompletions(pipeCloseCmd, nil,
		carapace.ActionValues().Usage("pipe name"),
	)

	pipeServerCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModulePipeServer) + " [action] [pipe_name]",
		Short: "Manage pipe server operations",
		Long:  "Start, stop, or list pipe servers for receiving data from clients.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PipeServerCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModulePipeServer,
			"ttp":    "T1090",
		},
		Example: `Pipe server operations:
  ~~~
  pipe server start \\.\pipe\mypipe       # Start a pipe server
  pipe server stop \\.\pipe\mypipe       # Stop a pipe server
  pipe server list               # List all running pipe servers
  pipe server status \\.\pipe\mypipe      # Check server status and cache size
  pipe server clear \\.\pipe\mypipe       # Clear cached data for a pipe
  ~~~`,
	}
	common.BindArgCompletions(pipeServerCmd, nil,
		carapace.ActionValues("start", "stop", "list", "clear", "status").Usage("action"),
		carapace.ActionValues().Usage("pipe name (required for start/stop/clear/status)"),
	)

	// Add subcommands to the main pipe command
	pipeCmd.AddCommand(pipeUploadCmd, pipeReadCmd, pipeServerCmd)
	// , pipeCloseCmd

	return []*cobra.Command{pipeCmd}
}

// Register registers all pipe-related commands.
func Register(con *repl.Console) {
	RegisterPipeUploadFunc(con)
	RegisterPipeReadFunc(con)
	RegisterPipeCloseFunc(con)
	RegisterPipeServerFunc(con)
}
