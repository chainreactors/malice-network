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
		Long:  help.GetHelpFor("sessions"),
		Run: func(cmd *cobra.Command, args []string) {
			SessionsCmd(cmd, con)
		},
	}
	common.Bind("sessions", true, sessionsCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "show all sessions")
	})

	noteCommand := &cobra.Command{
		Use:   consts.CommandNote,
		Short: "add note to session",
		Long:  help.GetHelpFor("note"),
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			noteCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(noteCommand,
		nil,
		common.SessionIDCompleter(con),
		carapace.ActionValues().Usage("session note name"),
	)

	groupCommand := &cobra.Command{
		Use:   consts.CommandGroup,
		Short: "group session",
		Long:  help.GetHelpFor("group"),
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			groupCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(groupCommand,
		nil,
		common.SessionIDCompleter(con),
		carapace.ActionValues().Usage("session group name"),
	)

	removeCommand := &cobra.Command{
		Use:   consts.CommandDelSession,
		Short: "del session",
		Long:  help.GetHelpFor("remove"),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			removeCmd(cmd, con)
		},
	}

	common.BindArgCompletions(removeCommand, nil, common.SessionIDCompleter(con))

	useCommand := &cobra.Command{
		Use:   consts.CommandUse,
		Short: "Use session",
		Long:  help.GetHelpFor("use"),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UseSessionCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(useCommand, nil, common.SessionIDCompleter(con))

	backCommand := &cobra.Command{
		Use:   consts.CommandBackgroup,
		Short: "back to root context",
		Long:  help.GetHelpFor(consts.CommandBackgroup),
		Run: func(cmd *cobra.Command, args []string) {
			con.ActiveTarget.Background()
			con.App.SwitchMenu(consts.ClientMenu)
			return
		},
	}

	observeCmd := &cobra.Command{
		Use:   consts.CommandObverse,
		Short: "observe session",
		Long:  help.GetHelpFor("observe"),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ObserveCmd(cmd, con)
		},
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
