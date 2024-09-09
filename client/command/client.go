package command

import (
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

func bindCommonCommands(bind bindFunc) {
	bind("",
		generic.Commands)

	bind(consts.GenericGroup,
		sessions.Commands,
		alias.Commands,
		extension.Commands,
		armory.Commands,
		mal.Commands,
	)

	bind(consts.ListenerGroup,
		listener.Commands,
	)
}

func BindClientsCommands(con *repl.Console) console.Commands {
	clientCommands := func() *cobra.Command {
		client := &cobra.Command{
			Short: "client commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}

		bind := makeBind(client, con)

		bindCommonCommands(bind)
		if con.ServerStatus == nil {
			err := generic.LoginCmd(&cobra.Command{}, con)
			if err != nil {
				con.Log.Errorf("Failed to login: %s", err)
				return nil
			}
		}
		return client
	}
	return clientCommands
}
