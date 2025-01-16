package command

import (
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/chainreactors/malice-network/client/command/action"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/config"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/mutant"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
)

func BindCommonCommands(bind BindFunc) {
	bind(consts.GenericGroup,
		generic.Commands)

	bind(consts.ManageGroup,
		sessions.Commands,
		alias.Commands,
		extension.Commands,
		armory.Commands,
		mal.Commands,
		config.Commands,
	)

	bind(consts.ListenerGroup,
		listener.Commands,
	)

	bind(consts.GeneratorGroup,
		build.Commands,
		action.Commands,
		mutant.Commands,
	)
}

func ConsoleRunnerCmd(con *repl.Console, cmd *cobra.Command) (pre, post func(cmd *cobra.Command, args []string) error) {
	common.Bind(cmd.Use, true, cmd, func(f *pflag.FlagSet) {
		f.String("auth", "", "auth token")
		f.Bool("console", false, "run console")
	})

	pre = func(cmd *cobra.Command, args []string) error {
		if cmd.Use == consts.CommandLogin || cmd.Use == consts.ClientMenu {
			return nil
		}
		return generic.LoginCmd(cmd, con)
	}

	// Close the RPC connection once exiting
	post = func(cmd *cobra.Command, _ []string) error {
		if run, _ := cmd.Flags().GetBool("console"); run || cmd.Use == consts.CommandLogin {
			return con.Start(BindClientsCommands, BindImplantCommands)
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

		bind := MakeBind(client, con)

		BindCommonCommands(bind)

		client.InitDefaultHelpCmd()
		client.InitDefaultHelpFlag()
		client.SetUsageFunc(help.UsageFunc)
		client.SetHelpFunc(help.HelpFunc)
		client.SetHelpCommandGroupID(consts.GenericGroup)

		RegisterClientFunc(con)
		RegisterImplantFunc(con)
		return client
	}
	return clientCommands
}

func RegisterClientFunc(con *repl.Console) {
	generic.Register(con)
	build.Register(con)
	action.Register(con)
	mutant.Register(con)
}
