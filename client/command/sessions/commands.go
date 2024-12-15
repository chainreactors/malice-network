package sessions

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	sessionsCmd := &cobra.Command{
		Use:   consts.CommandSessions,
		Short: "List and Choice sessions",
		Long: `Display a table of active sessions on the server, 
allowing you to navigate up and down to select a desired session. 
Press the Enter key to use the selected session. 
Use the -a or --all option to display all sessions, including those that have been disconnected.
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return SessionsCmd(cmd, con)
		},
		Example: `~~~
// List all active sessions
sessions

// List all sessions, including those that have been disconnected
sessions -a
~~~`,
	}

	common.Bind("sessions", true, sessionsCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "show all sessions")
	})

	sessCmd := &cobra.Command{
		Use:   consts.CommandSession,
		Short: "Session manager",
	}

	bindSessNewCmd := &cobra.Command{
		Use:   consts.CommandNewBindSession + " [session]",
		Short: "new bind session",
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewBindSessionCmd(cmd, con)
		},
	}

	common.BindFlag(bindSessNewCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "session name")
		f.StringP("target", "t", "", "session target")
		f.String("pipeline", "", "pipeline id")
		bindSessNewCmd.MarkFlagRequired("target")
		bindSessNewCmd.MarkFlagRequired("pipeline")
	})

	common.BindFlagCompletions(bindSessNewCmd, func(comp carapace.ActionMap) {
		comp["pipeline"] = common.AllPipelineCompleter(con)
	})

	noteCommand := &cobra.Command{
		Use:   consts.CommandSessionNote + " [note] [session]",
		Short: "add note to session",
		Long: `Add a note to a session. If a note already exists, it will be updated. 
When using an active session, only provide the new note.`,
		Args: cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			noteCmd(cmd, con)
			return
		},
		Example: `~~~
// Add a note to specified session
note newNote 08d6c05a21512a79a1dfeb9d2a8f262f

// Add a note when using an active session
note newNote
~~~`,
	}

	common.BindArgCompletions(noteCommand,
		nil,
		common.SessionIDCompleter(con),
		carapace.ActionValues().Usage("session note name"),
	)

	groupCommand := &cobra.Command{
		Use:   consts.CommandSessionGroup + " [group] [session]",
		Short: "group session",
		Long: `Add a session to a group. If the group does not exist, it will be created.
When using an active session, only provide the group name.`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return groupCmd(cmd, con)
		},
		Example: `~~~
// Add a session to a group
group newGroup 08d6c05a21512a79a1dfeb9d2a8f262f

// Add a session to a group when using an active session
group newGroup
~~~`,
	}

	common.BindArgCompletions(groupCommand,
		nil,
		common.SessionIDCompleter(con),
		carapace.ActionValues().Usage("session group name"),
	)

	removeCommand := &cobra.Command{
		Use:   consts.CommandRemoveSession + " [session]",
		Short: "remove session",
		Long:  "Remove a specified session.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeCmd(cmd, con)
		},
		Example: `~~~
// remove a specified session
remove 08d6c05a21512a79a1dfeb9d2a8f262f
~~~`,
	}

	common.BindArgCompletions(removeCommand, nil, common.SessionIDCompleter(con))

	sessCmd.AddCommand(bindSessNewCmd, noteCommand, groupCommand, removeCommand)
	useCommand := &cobra.Command{
		Use:   consts.CommandUse + " [session]",
		Short: "Use session",
		Long:  "use",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UseSessionCmd(cmd, con)
		},
	}

	common.BindArgCompletions(useCommand, nil, common.SessionIDCompleter(con))

	backCommand := &cobra.Command{
		Use:   consts.CommandBackground,
		Short: "back to root context",
		Long:  "Exit the current session and return to the root context.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return BackGround(cmd, con)
		},
	}

	observeCmd := &cobra.Command{
		Use:   consts.CommandObverse,
		Short: "observe manager",
		Long:  "Control observers to listen session in the background.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ObserveCmd(cmd, con)
		},
		Example: `~~~
// List all observers
observe -l

// Remove observer
observe -r
~~~`,
	}

	common.BindFlag(observeCmd, func(f *pflag.FlagSet) {
		f.BoolP("list", "l", false, "list all observers")
		f.BoolP("remove", "r", false, "remove observer")
	})

	common.BindArgCompletions(observeCmd, nil, common.SessionIDCompleter(con))

	historyCommand := &cobra.Command{
		Use:   consts.CommandHistory,
		Short: "show log history",
		Long:  "Displays the specified number of log lines of the current session.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return historyCmd(cmd, con)
		},
	}

	common.BindArgCompletions(historyCommand, nil, carapace.ActionValues().Usage("number of lines"))

	infoCommand := &cobra.Command{
		Use:   "info",
		Short: "show session info",
		Long:  "Displays the specified session info.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return SessionInfoCmd(cmd, con)
		},
	}
	sessionsCmd.AddCommand(infoCommand)

	return []*cobra.Command{
		sessionsCmd,
		sessCmd,
		backCommand,
		useCommand,
		observeCmd,
		historyCommand,
	}
}
