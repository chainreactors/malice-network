package command

import (
	"github.com/chainreactors/grumble"
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

	bind(consts.GenericGroup,
		login.Command,
		sessions.Command,
		use.Command,
		listener.Commands,
		alias.Commands,
	)

	login.LoginCmd(&grumble.Context{}, con)
}
