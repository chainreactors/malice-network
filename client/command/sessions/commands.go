package sessions

import (
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/command/flags"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *console.Console) []*cobra.Command {
	sessionsCmd := &cobra.Command{
		Use:   "sessions",
		Short: "List sessions",
		Long:  help.GetHelpFor("sessions"),
		Run: func(cmd *cobra.Command, args []string) {
			SessionsCmd(cmd, con)
		},
		GroupID: consts.GenericGroup,
	}
	flags.Bind("sessions", true, sessionsCmd, func(f *pflag.FlagSet) {
		f.Bool("all", false, "show all sessions")
	})

	noteCommand := &cobra.Command{
		Use:   "note",
		Short: "add note to session",
		Long:  help.GetHelpFor("note"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			noteCmd(cmd, con)
			return
		},
		GroupID: consts.GenericGroup,
	}

	carapace.Gen(noteCommand).PositionalCompletion(
		carapace.ActionValues().Usage("session note name"),
	)

	flags.Bind("note", false, noteCommand, func(f *pflag.FlagSet) {
		f.String("id", "", "session id")
	})

	flags.BindFlagCompletions(noteCommand, func(comp *carapace.ActionMap) {
		(*comp)["id"] = completer.BasicSessionIDCompleter(con)
	})

	groupCommand := &cobra.Command{
		Use:   "group",
		Short: "group session",
		Long:  help.GetHelpFor("group"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			groupCmd(cmd, con)
			return
		},
		GroupID: consts.GenericGroup,
	}

	carapace.Gen(groupCommand).PositionalCompletion(
		carapace.ActionValues().Usage("session group name"),
	)

	flags.Bind("group", false, groupCommand, func(f *pflag.FlagSet) {
		f.String("id", "", "session id")
	})

	flags.BindFlagCompletions(groupCommand, func(comp *carapace.ActionMap) {
		(*comp)["id"] = completer.BasicSessionIDCompleter(con)
	})

	removeCommand := &cobra.Command{
		Use:   "remove",
		Short: "remove session",
		Long:  help.GetHelpFor("remove"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			removeCmd(cmd, con)
		},
		GroupID: consts.GenericGroup,
	}

	carapace.Gen(removeCommand).PositionalCompletion(
		completer.BasicSessionIDCompleter(con),
	)

	return []*cobra.Command{
		sessionsCmd,
		noteCommand,
		groupCommand,
		removeCommand,
	}
}
