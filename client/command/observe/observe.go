package observe

import (
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/command/flags"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Command(con *console.Console) []*cobra.Command {
	observeCmd := &cobra.Command{
		Use:   "observe",
		Short: "observe session",
		Long:  help.GetHelpFor("observe"),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ObserveCmd(cmd, con)
		},
	}
	flags.Bind("observe", true, observeCmd, func(f *pflag.FlagSet) {
		f.BoolP("list", "l", false, "list all observers")
		f.BoolP("remove", "r", false, "remove observer")
	})

	carapace.Gen(observeCmd).PositionalCompletion(
		completer.BasicSessionIDCompleter(con),
	)
	return []*cobra.Command{}
}

func ObserveCmd(cmd *cobra.Command, con *console.Console) {
	var session *clientpb.Session
	isList, _ := cmd.Flags().GetBool("list")
	if isList {
		for i, ob := range con.Observers {
			console.Log.Infof("%d: %s", i, ob.SessionId())
		}
		return
	}

	idArg := cmd.Flags().Args()
	if idArg == nil {
		if con.GetInteractive() != nil {
			idArg = []string{con.GetInteractive().SessionId}
		} else {
			for i, ob := range con.Observers {
				console.Log.Infof("%d: %s", i, ob.SessionId())
			}
			return
		}
	}
	for _, sid := range idArg {
		session = con.Sessions[sid]

		if session == nil {
			console.Log.Warn(console.ErrNotFoundSession.Error())
		}
		isRemove, _ := cmd.Flags().GetBool("remove")
		if isRemove {
			con.RemoveObserver(session.SessionId)
		} else {
			con.AddObserver(session)
		}
	}
}
