package cli

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func rootCmd(con *core.Console) (*cobra.Command, error) {
	var cmd = &cobra.Command{
		Use: "client",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := generic.LoginCmd(cmd, con); err != nil {
				return err
			}
			return con.Start(command.BindClientsCommands, command.BindImplantCommands)
		},
	}
	cmd.TraverseChildren = true

	// 添加 --mcp flag
	cmd.PersistentFlags().String("mcp", "", "enable MCP server with address (e.g., 127.0.0.1:5005)")
	// 添加 --rpc flag
	cmd.PersistentFlags().String("rpc", "", "enable local gRPC server with address (e.g., 127.0.0.1:15004)")
	// 添加全局 --wizard flag
	command.RegisterWizardFlag(cmd)

	bind := command.MakeBind(cmd, con, "golang")
	command.BindCommonCommands(bind)
	// 包装 PersistentPreRunE 以支持 wizard 模式
	originalPre, originalPost := command.ConsoleRunnerCmd(con, cmd)
	cmd.PersistentPreRunE, cmd.PersistentPostRunE = command.WrapWithWizardSupport(con, originalPre, originalPost)
	cmd.AddCommand(command.ImplantCmd(con))
	carapace.Gen(cmd)

	return cmd, nil
}
