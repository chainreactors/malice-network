package command

import (
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/login"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/use"
	"github.com/chainreactors/malice-network/client/command/version"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func BindClientsCommands(con *console.Console) {
	bind := makeBind(con)

	bind("",
		version.Command)

	bind(consts.AliasesGroup)
	bind(consts.ExtensionGroup)
	bind(consts.GenericGroup,
		login.Command,
		sessions.Command,
		use.Command,
		listener.Commands,
		alias.Commands,
	)

	//certCmd := &grumble.Command{
	//	Name: "cert",
	//	Help: "Register cert from server",
	//	Flags: func(f *grumble.Flags) {
	//		f.String("", "host", "", "Host to register")
	//		f.String("u", "user", "test", "User to register")
	//		f.Int("p", "port", 40000, "Port to register")
	//	},
	//	Run: func(ctx *grumble.Context) error {
	//		cert.CertCmd(ctx, con)
	//		return nil
	//	},
	//}
	//con.App.AddCommand(certCmd)

}
