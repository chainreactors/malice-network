package common

import "github.com/spf13/cobra"

// RemoveCommandByName removes all commands with the given name from parent.
// Collects targets first to avoid slice mutation during iteration.
func RemoveCommandByName(parent *cobra.Command, name string) {
	var toRemove []*cobra.Command
	for _, cmd := range parent.Commands() {
		if cmd.Name() == name {
			toRemove = append(toRemove, cmd)
		}
	}
	for _, cmd := range toRemove {
		parent.RemoveCommand(cmd)
	}
}

// RemoveCommandsByGroup removes all commands belonging to the given group from parent.
// Collects targets first to avoid slice mutation during iteration.
func RemoveCommandsByGroup(parent *cobra.Command, groupID string) {
	var toRemove []*cobra.Command
	for _, cmd := range parent.Commands() {
		if cmd.GroupID == groupID {
			toRemove = append(toRemove, cmd)
		}
	}
	for _, cmd := range toRemove {
		parent.RemoveCommand(cmd)
	}
}
