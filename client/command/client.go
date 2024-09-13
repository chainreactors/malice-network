package command

import (
	"github.com/chainreactors/malice-network/client/assets"
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
			repl.NewConfigLogin(con, args[0])
		} else {
			err := generic.LoginCmd(nil, con)
			if err != nil {
				con.Log.Errorf("Failed to login: %s", err)
				return nil
			}
		}

		for _, malName := range assets.GetInstalledMalManifests() {
			manifest, err := mal.LoadMalManiFest(con, malName)
			// Absorb error in case there's no extensions manifest
			if err != nil {
				//con doesn't appear to be initialised here?
				//con.PrintErrorf("Failed to load extension: %s", err)
				repl.Log.Errorf("Failed to load mal: %s\n", err)
				continue
			}

			if _, err := con.Plugins.LoadPlugin(manifest, con); err == nil {
				//plugin.GenerateLuaDefinitionFile(plug.LuaVM, "lua.lua")
			} else {
				repl.Log.Errorf("Failed to load mal: %s\n", err)
				continue
			}
		}
		RegisterImplantFunc(con)
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
		return client
	}
	return clientCommands
}
