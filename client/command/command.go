package command

import (
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

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
				if c.Annotations == nil {
					c.Annotations = map[string]string{}
				}
				c.Annotations["menu"] = cmd.Name()
				c.GroupID = group
				if cmd.Name() == consts.ImplantMenu {
					updateCommand(con, c, group)
				}
				cmd.AddCommand(c)
			}
		}
	}
}

func updateCommand(con *repl.Console, c *cobra.Command, group string) {
	c.SetHelpFunc(help.HelpFunc)
	c.SetUsageFunc(help.UsageFunc)
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	help.RenderOpsec(c.Annotations["opsec"], c.Use)
	con.CMDs[c.Name()] = c

	for _, subCmd := range c.Commands() {
		updateCommand(con, subCmd, group)
	}
}
