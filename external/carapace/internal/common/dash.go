package common

import (
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

// IsDash checks if command contains a dash disabling flag parsing
//
//	example action positional1 -- dash1 dash2
func IsDash(cmd *cobra.Command) bool {
	return slices.Contains(cmd.Flags().Args(), "--")
}

func IndexDash(cmd *cobra.Command) int {
	return slices.Index(cmd.Flags().Args(), "--")
}
