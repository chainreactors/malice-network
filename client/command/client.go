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
	"google.golang.org/grpc"
)

func bindCommonCommands(bind bindFunc) {
	bind(consts.GenericGroup,
		generic.Commands)

	bind(consts.ManageGroup,
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

func ConsoleCmd(con *repl.Console) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: "Start the client console",
	}

	consoleCmd.RunE, consoleCmd.PersistentPostRunE = ConsoleRunnerCmd(con, true)
	return consoleCmd
}

func ConsoleRunnerCmd(con *repl.Console, run bool) (pre, post func(cmd *cobra.Command, args []string) error) {
	var ln *grpc.ClientConn

	pre = func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			err := repl.NewConfigLogin(con, args[0])
			if err != nil {
				return nil
			}
		} else {
			err := generic.LoginCmd(nil, con)
			if err != nil {
				return nil
			}
		}

		return con.Start(BindClientsCommands, BindImplantCommands)
	}

	// Close the RPC connection once exiting
	post = func(_ *cobra.Command, _ []string) error {
		if ln != nil {
			return ln.Close()
		}

		return nil
	}

	return pre, post
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

		client.InitDefaultHelpCmd()
		client.SetHelpCommandGroupID(consts.GenericGroup)
		return client
	}
	return clientCommands
}
