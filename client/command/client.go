package command

import (
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/login"
	"github.com/chainreactors/malice-network/client/command/observe"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/command/use"
	"github.com/chainreactors/malice-network/client/command/version"
	cc "github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

func BindClientsCommands(con *cc.Console) console.Commands {
	clientCommands := func() *cobra.Command {
		client := &cobra.Command{
			Short: "client commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}
		bind := makeBind(client, con)

		bind("",
			version.Command)

		bind(consts.GenericGroup,
			login.Command,
			sessions.Commands,
			use.Command,
			tasks.Command,
			alias.Commands,
			extension.Commands,
			armory.Commands,
			observe.Command,
			explorer.Commands,
		)

		bind(consts.ListenerGroup,
			listener.Commands,
		)

		if con.ServerStatus == nil {
			err := login.LoginCmd(&cobra.Command{}, con)
			if err != nil {
				cc.Log.Errorf("Failed to login: %s", err)
				return nil
			}
		}
		return client
	}
	return clientCommands
}
