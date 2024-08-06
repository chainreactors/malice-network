package command

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/basic"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/jobs"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/login"
	"github.com/chainreactors/malice-network/client/command/observe"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/command/use"
	"github.com/chainreactors/malice-network/client/command/version"
	"github.com/chainreactors/malice-network/client/command/website"
	"github.com/chainreactors/malice-network/client/console"

	"github.com/chainreactors/malice-network/helper/consts"
)

func BindClientsCommands(con *console.Console) {
	bind := makeBind(con)

	bind("",
		version.Command)
	bind(consts.SessionGroup,
		basic.Commands,
	)

	bind(consts.GenericGroup,
		login.Command,
		sessions.Command,
		use.Command,
		tasks.Command,
		jobs.Command,
		listener.Commands,
		alias.Commands,
		armory.Commands,
		observe.Command,
		website.Commands,
		explorer.Commands,
	)
	if con.ServerStatus == nil {
		login.LoginCmd(&grumble.Context{}, con)
	}
}
