package common

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var stdinIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func ShouldUseStaticOutput(cmd *cobra.Command) bool {
	if cmd != nil && cmd.Flags() != nil && cmd.Flags().Lookup("static") != nil {
		isStatic, err := cmd.Flags().GetBool("static")
		if err == nil && isStatic {
			return true
		}
	}

	return !stdinIsTerminal()
}
