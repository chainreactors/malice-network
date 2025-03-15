package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// ExecuteShell returns a cobra command to execute a line through the system shell.
// This uses the os/exec package to execute the command. The default command name is `!`.
func ExecuteShell() *cobra.Command {
	shellCmd := &cobra.Command{
		Use:                "!",
		Short:              "Execute the remaining arguments with system shell",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			// Only at least one argument is needed, since the command
			// itself is stripped before those args are passed as an array.
			if len(args) < 1 {
				return errors.New("command requires one or more arguments")
			}

			// Above, the length of args is checked to be at least 2
			path, err := exec.LookPath(args[0])
			if err != nil {
				return err
			}

			shellCmd := exec.Command(path, args[1:]...)

			// Load OS environment
			shellCmd.Env = os.Environ()

			out, err := shellCmd.CombinedOutput()
			if err != nil {
				return err
			}

			fmt.Print(string(out))

			return nil
		},
	}

	return shellCmd
}
