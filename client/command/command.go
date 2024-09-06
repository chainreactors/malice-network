package command

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

const defaultTimeout = 60

// Bind is a convenience function to bind flags to a given command.
// name - The name of the flag set (can be empty).
// cmd  - The command to which the flags should be bound.

type bindFunc func(group string, cmds ...func(con *repl.Console) []*cobra.Command)

// BindFlagCompletions is a convenience function for adding completions to a command's flags.
// cmd - The command owning the flags to complete.
// bind - A function exposing a map["flag-name"]carapace.Action.
//func BindFlagCompletions(cmd *cobra.Command, bind func(comp *carapace.ActionMap)) {
//	comps := make(carapace.ActionMap)
//	bind(&comps)
//
//	carapace.Gen(cmd).FlagCompletion(comps)
//}

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
				cmd.AddCommand(c)
			}
		}
	}
}

// commandBinder is a helper used to bind commands to a given menu, for a given "command help group".
//
// @group - Name of the group under which the command should be shown. Preferably use a string in the constants package.
// @ cmds - A list of functions returning a list of root commands to bind. See any package's `commands.go` file and function.
type commandBinder func(group string, cmds ...func(con *repl.Console) []*grumble.Command)
