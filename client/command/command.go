package command

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
)

const defaultTimeout = 60

// Bind is a convenience function to bind flags to a given command.
// name - The name of the flag set (can be empty).
// cmd  - The command to which the flags should be bound.
// flags - A function exposing the flag set through which flags are declared.
func Bind(name string, persistent bool, cmd *grumble.Command, flags func(f *grumble.Flags)) {
	//flagSet := func(f *grumble.Flags) {
	//
	//}              // Create the flag set.
	//flags(flagSet) // Let the user bind any number of flags to it.
	//
	//if persistent {
	//	cmd.PersistentFlags().AddFlagSet(flagSet)
	//} else {
	//	cmd.Flags().AddFlagSet(flagSet)
	//}
}

// BindFlagCompletions is a convenience function for adding completions to a command's flags.
// cmd - The command owning the flags to complete.
// bind - A function exposing a map["flag-name"]carapace.Action.
//func BindFlagCompletions(cmd *cobra.Command, bind func(comp *carapace.ActionMap)) {
//	comps := make(carapace.ActionMap)
//	bind(&comps)
//
//	carapace.Gen(cmd).FlagCompletion(comps)
//}

// makeBind returns a commandBinder helper function
// @menu  - The command menu to which the commands should be bound (either server or implant menu).
func makeBind(con *console.Console) func(group string, cmds ...func(con *console.Console) []*grumble.Command) {
	return func(group string, cmds ...func(con *console.Console) []*grumble.Command) {
		var grp *grumble.Group
		if group != "" {
			grp = con.App.Groups().Find(group)

			if grp == nil {
				grp = grumble.NewGroup(group)
				con.App.AddGroup(grp)
			}
		}

		// Bind the command to the root
		for _, command := range cmds {
			for _, c := range command(con) {
				if group == "" {
					con.App.AddCommand(c)
				} else {
					grp.AddCommand(c)
				}
			}
		}
	}
}

// commandBinder is a helper used to bind commands to a given menu, for a given "command help group".
//
// @group - Name of the group under which the command should be shown. Preferably use a string in the constants package.
// @ cmds - A list of functions returning a list of root commands to bind. See any package's `commands.go` file and function.
type commandBinder func(group string, cmds ...func(con *console.Console) []*grumble.Command)
