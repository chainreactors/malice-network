package common

import (
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

func ShouldSuppressStartupOutput(cmd *cobra.Command) bool {
	return !ShouldStartConsole(cmd)
}
