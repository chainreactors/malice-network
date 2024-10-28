package command

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"path/filepath"
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

	bind(consts.GeneratorGroup,
		build.Commands,
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
			filename := args[0]
			if !file.Exist(filename) {
				if cfg, err := mtls.ReadConfig(filepath.Join(assets.GetConfigDir(), filepath.Base(filename))); err == nil {
					err = repl.Login(con, cfg)
					if err != nil {
						return nil
					}
				} else {
					con.Log.Warnf("not found file, maybe %s already move to config path", filename)
					err := generic.LoginCmd(nil, con)
					if err != nil {
						return nil
					}
				}
			} else {
				err := repl.NewConfigLogin(con, filename)
				if err != nil {
					core.Log.Errorf("Error logging in: %s", err)
					return nil
				}
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
			Use:   "client",
			Short: "client commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}

		bind := makeBind(client, con)

		bindCommonCommands(bind)

		client.InitDefaultHelpCmd()
		client.InitDefaultHelpFlag()
		client.SetHelpCommandGroupID(consts.GenericGroup)
		RegisterClientFunc(con)
		RegisterImplantFunc(con)
		return client
	}
	return clientCommands
}

func RegisterClientFunc(con *repl.Console) {
	generic.Register(con)
}
