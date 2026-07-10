package command

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

// Bind is a convenience function to bind flags to a given command.
// name - The name of the flag set (can be empty).
// cmd  - The command to which the flags should be bound.

type BindFunc func(group string, cmds ...func(con *core.Console) []*cobra.Command)

func MakeBind(cmd *cobra.Command, con *core.Console, source string) BindFunc {
	return func(group string, cmds ...func(con *core.Console) []*cobra.Command) {
		common.RegisterCommandGroup(cmd, group)
		// Bind the command to the root
		for _, command := range cmds {
			for _, c := range command(con) {
				common.RegisterCommand(cmd, con, group, source, c)
			}
		}
	}
}
