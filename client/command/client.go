package command

import (
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func bindCommonCommands(bind bindFunc) {
	bind("",
		generic.Commands)

	bind(consts.GenericGroup,
		sessions.Commands,
		tasks.Command,
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
		common.Bind("common flag", true, client, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", consts.DefaultTimeout, "command timeout in seconds")
		})

		bind := makeBind(client, con)

		bindCommonCommands(bind)
		if con.ServerStatus == nil {
			err := generic.LoginCmd(&cobra.Command{}, con)
			if err != nil {
				repl.Log.Errorf("Failed to login: %s", err)
				return nil
			}
		}
		return client
	}
	return clientCommands
}
