package common

import (
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/spf13/cobra"
)

// ShouldStartConsole reports whether the current command invocation should
// enter the interactive console/REPL after login.
func ShouldStartConsole(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}

	run, _ := cmd.Flags().GetBool("console")
	if run {
		return true
	}

	if !StdinIsTerminal() {
		return false
	}

	return cmd == cmd.Root() || cmd.Use == consts.CommandLogin
}

func ShouldStartDaemon(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}

	run, _ := cmd.Flags().GetBool("daemon")
	return run
}

func ShouldStartRuntime(cmd *cobra.Command) bool {
	return ShouldStartConsole(cmd) || ShouldStartDaemon(cmd)
}

func ShouldSuppressStartupOutput(cmd *cobra.Command) bool {
	return !ShouldStartRuntime(cmd)
}

func ValidateExecutionModeFlags(cmd *cobra.Command) error {
	if cmd == nil {
		return nil
	}

	runConsole, _ := cmd.Flags().GetBool("console")
	runDaemon, _ := cmd.Flags().GetBool("daemon")
	if runConsole && runDaemon {
		return fmt.Errorf("--console and --daemon cannot be used together")
	}

	return nil
}
