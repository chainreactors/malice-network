package sessions

import (
	"github.com/chainreactors/malice-network/client/command/common"
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
	}
	common.Bind("sessions", true, sessionsCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "show all sessions")
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
	}

	carapace.Gen(noteCommand).PositionalCompletion(
		carapace.ActionValues().Usage("session note name"),
	)

	common.Bind("note", false, noteCommand, func(f *pflag.FlagSet) {
		f.String("id", "", "session id")
	})

	common.BindFlagCompletions(noteCommand, func(comp *carapace.ActionMap) {
		(*comp)["id"] = common.BasicSessionIDCompleter(con)
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
	}

	carapace.Gen(groupCommand).PositionalCompletion(
		carapace.ActionValues().Usage("session group name"),
	)

	common.Bind("group", false, groupCommand, func(f *pflag.FlagSet) {
		f.String("id", "", "session id")
	})

	common.BindFlagCompletions(groupCommand, func(comp *carapace.ActionMap) {
		(*comp)["id"] = common.BasicSessionIDCompleter(con)
	})

	removeCommand := &cobra.Command{
		Use:   "remove",
		Short: "remove session",
		Long:  help.GetHelpFor("remove"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			removeCmd(cmd, con)
		},
	}

	carapace.Gen(removeCommand).PositionalCompletion(
		common.BasicSessionIDCompleter(con),
	)

	useCommand := &cobra.Command{
		Use:   "use",
		Short: "Use session",
		Long:  help.GetHelpFor("use"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UseSessionCmd(cmd, con)
			return
		},
	}

	carapace.Gen(useCommand).PositionalCompletion(
		common.BasicSessionIDCompleter(con),
	)

	backCommand := &cobra.Command{
		Use:   "background",
		Short: "back to root context",
		Long:  help.GetHelpFor("background"),
		Run: func(cmd *cobra.Command, args []string) {
			con.ActiveTarget.Background()
			con.App.SwitchMenu(consts.ClientMenu)
			return
		},
	}

	return []*cobra.Command{
		sessionsCmd,
		noteCommand,
		groupCommand,
		removeCommand,
		backCommand,
		useCommand,
	}
}
