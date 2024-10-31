package command

import (
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

const defaultTimeout = 60

// Bind is a convenience function to bind flags to a given command.
// name - The name of the flag set (can be empty).
// cmd  - The command to which the flags should be bound.

type bindFunc func(group string, cmds ...func(con *repl.Console) []*cobra.Command)

func makeBind(cmd *cobra.Command, con *repl.Console) bindFunc {
	return func(group string, cmds ...func(con *repl.Console) []*cobra.Command) {
		found := false

		// Ensure the given command group is available in the menu.
		if group != "" {
			for _, grp := range cmd.Groups() {
				if grp.Title == group {
					found = true
					break
				}
			}

			if !found {
				cmd.AddGroup(&cobra.Group{
					ID:    group,
					Title: group,
				})
			}
		}

		// Bind the command to the root
		for _, command := range cmds {
			for _, c := range command(con) {
				c.GroupID = group
				c.SetHelpFunc(help.HelpFunc)
				c.SetUsageFunc(help.UsageFunc)
				SetColoredUse(c)

				if c.Annotations == nil {
					c.Annotations = map[string]string{"menu": cmd.Name()}
				} else {
					c.Annotations["menu"] = cmd.Name()
				}
				cmd.AddCommand(c)
				con.CMDs[c.Name()] = c
			}
		}
	}
}
