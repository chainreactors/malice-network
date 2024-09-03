package use

import (
	"github.com/chainreactors/malice-network/client/command/completer"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Command(con *console.Console) []*cobra.Command {
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
		completer.BasicSessionIDCompleter(con),
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
		useCommand,
		backCommand,
	}
}

func UseSessionCmd(cmd *cobra.Command, con *console.Console) {
	var session *clientpb.Session
	err := con.UpdateSessions(false)
	if err != nil {
		console.Log.Errorf("%s", err)
		return
	}
	idArg := cmd.Flags().Arg(0)
	if idArg != "" {
		session = con.Sessions[idArg]
	}

	if session == nil {
		console.Log.Errorf(console.ErrNotFoundSession.Error())
		return
	}

	con.ActiveTarget.Set(session)
	con.App.SwitchMenu(consts.ImplantMenu)
	console.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
}
