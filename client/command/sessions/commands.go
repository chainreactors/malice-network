package sessions

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	sessionsCmd := &cobra.Command{
		Use:   consts.CommandSessions,
		Short: "List sessions",
		Long: help.FormatLongHelp(`Display a table of active sessions on the server, 
allowing you to navigate up and down to select a desired session. 
Press the Enter key to use the selected session. 
Use the -a or --all option to display all sessions, including those that have been disconnected.
		`),
		Run: func(cmd *cobra.Command, args []string) {
			SessionsCmd(cmd, con)
		},
		Example: help.FormatLongHelp(`~~~
// List all active sessions
sessions

// List all sessions, including those that have been disconnected
sessions -a
			~~~`),
	}

	common.Bind("sessions", true, sessionsCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "show all sessions")
	})

	noteCommand := &cobra.Command{
		Use:   consts.CommandNote + " [note] [session]",
		Short: "add note to session",
		Long: help.FormatLongHelp(`Add a note to a session. If a note already exists, it will be updated. 
When using an active session, only provide the new note.`),
		Args: cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			noteCmd(cmd, con)
			return
		},
		Example: help.FormatLongHelp(`~~~
// Add a note to specified session
note newNote 08d6c05a21512a79a1dfeb9d2a8f262f

// Add a note when using an active session
note newNote
			~~~`),
	}

	common.BindArgCompletions(noteCommand,
		nil,
		common.SessionIDCompleter(con),
		carapace.ActionValues().Usage("session note name"),
	)

	groupCommand := &cobra.Command{
		Use:   consts.CommandGroup + " [group] [session]",
		Short: "group session",
		Long: help.FormatLongHelp(`Add a session to a group. If the group does not exist, it will be created.
When using an active session, only provide the group name.`),
		Args: cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			groupCmd(cmd, con)
			return
		},
		Example: help.FormatLongHelp(`~~~
// Add a session to a group
group newGroup 08d6c05a21512a79a1dfeb9d2a8f262f

// Add a session to a group when using an active session
group newGroup
			~~~`),
	}

	common.BindArgCompletions(groupCommand,
		nil,
		common.SessionIDCompleter(con),
		carapace.ActionValues().Usage("session group name"),
	)

	removeCommand := &cobra.Command{
		Use:   consts.CommandDelSession + " [session]",
		Short: "del session",
		Long:  help.FormatLongHelp("Del a specified session."),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			removeCmd(cmd, con)
		},
		Example: help.FormatLongHelp(`~~~
// Delete a specified session
del 08d6c05a21512a79a1dfeb9d2a8f262f
			~~~`),
	}

	common.BindArgCompletions(removeCommand, nil, common.SessionIDCompleter(con))

	useCommand := &cobra.Command{
		Use:   consts.CommandUse + " [session]",
		Short: "Use session",
		Long:  help.FormatLongHelp("use"),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UseSessionCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(useCommand, nil, common.SessionIDCompleter(con))

	backCommand := &cobra.Command{
		Use:   consts.CommandBackground,
		Short: "back to root context",
		Long:  help.FormatLongHelp("Exit the current session and return to the root context."),
		Run: func(cmd *cobra.Command, args []string) {
			con.ActiveTarget.Background()
			con.App.SwitchMenu(consts.ClientMenu)
			return
		},
	}

	observeCmd := &cobra.Command{
		Use:   consts.CommandObverse,
		Short: "observe manager",
		Long:  help.FormatLongHelp("Control observers to listen session in the background."),
		Run: func(cmd *cobra.Command, args []string) {
			ObserveCmd(cmd, con)
		},
		Example: help.FormatLongHelp(`~~~
// List all observers
observe -l

// Remove observer
observe -r
			~~~`),
	}

	common.BindFlag(observeCmd, func(f *pflag.FlagSet) {
		f.BoolP("list", "l", false, "list all observers")
		f.BoolP("remove", "r", false, "remove observer")
	})

	common.BindArgCompletions(observeCmd, nil, common.SessionIDCompleter(con))

	return []*cobra.Command{
		sessionsCmd,
		noteCommand,
		groupCommand,
		removeCommand,
		backCommand,
		useCommand,
		observeCmd,
	}
}
