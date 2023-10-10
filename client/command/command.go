package command

import (
	"github.com/chainreactors/malice-network/client/command/cert"
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

	certCmd := &grumble.Command{
		Name: "cert",
		Help: "Register cert from server",
		Flags: func(f *grumble.Flags) {
			f.String("h", "host", "", "Host to register")
			f.String("u", "user", "test", "User to register")
		},
		Run: func(ctx *grumble.Context) error {
			cert.CertCmd(ctx, con)
			return nil
		},
	}

	con.App.AddCommand(verCmd)
	con.App.AddCommand(certCmd)
}
