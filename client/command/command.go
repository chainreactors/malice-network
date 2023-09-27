package command

import (
	"github.com/chainreactors/malice-network/client/command/version"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/desertbit/grumble"
)

func BindCommands(con *console.Console) {

	verCmd := &grumble.Command{
		Name: "version",
		Help: "List current aliases",
		//LongHelp: help.GetHelpFor([]string{consts.AliasesStr}),
		Run: func(ctx *grumble.Context) error {
			//con.Println()
			version.VersionCmd(ctx, con)
			//con.Println()
			return nil
		},
		//HelpGroup: consts.GenericHelpGroup,
	}
	con.App.AddCommand(verCmd)

}
