package command

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/ai"
	"github.com/chainreactors/malice-network/client/command/audit"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/command/cert"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/config"
	"github.com/chainreactors/malice-network/client/command/context"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/mutant"
	"github.com/chainreactors/malice-network/client/command/pipeline"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/website"
	"github.com/chainreactors/malice-network/client/command/wizard"
)

func BindCommonCommands(bind BindFunc) {
	bind(consts.GenericGroup,
		generic.Commands,
		ai.Commands,
		wizard.Commands)

	bind(consts.ManageGroup,
		sessions.Commands,
		alias.Commands,
		extension.Commands,
		armory.Commands,
		mal.Commands,
		config.Commands,
		context.Commands,
		cert.Commands,
		audit.Commands,
	)

	bind(consts.ListenerGroup,
		listener.Commands,
		website.Commands,
		pipeline.Commands,
	)

	bind(consts.GeneratorGroup,
		build.Commands,
		mutant.Commands,
	)
}

func ConsoleRunnerCmd(con *core.Console, cmd *cobra.Command) (pre, post func(cmd *cobra.Command, args []string) error) {
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

func BindClientsCommands(con *core.Console) console.Commands {
	clientCommands := func() *cobra.Command {
		client := &cobra.Command{
			Use:   "client",
			Short: "client commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}

		// 注册全局 --wizard 标志
		RegisterWizardFlag(client)
		// 包装 PersistentPreRunE 以支持 wizard 模式
		client.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			return HandleWizardFlag(cmd, con)
		}

		bind := MakeBind(client, con, "golang")

		BindCommonCommands(bind)

		client.InitDefaultHelpCmd()
		client.InitDefaultHelpFlag()
		client.SetUsageFunc(help.UsageFunc)
		client.SetHelpFunc(help.HelpFunc)
		client.SetHelpCommandGroupID(consts.GenericGroup)

		// 为根命令注册 carapace 补全（使 PersistentFlags 在子命令中显示）
		carapace.Gen(client)

		RegisterClientFunc(con)
		RegisterImplantFunc(con)
		return client
	}
	return clientCommands
}

func RegisterClientFunc(con *core.Console) {
	generic.Register(con)
	build.Register(con)
	mutant.Register(con)
	context.Register(con)
	common.Register(con)
	website.Register(con)
}
