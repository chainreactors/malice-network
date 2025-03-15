package cli

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func rootCmd(con *repl.Console) (*cobra.Command, error) {
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
	bind := command.MakeBind(cmd, con)
	command.BindCommonCommands(bind)
	cmd.PersistentPreRunE, cmd.PersistentPostRunE = command.ConsoleRunnerCmd(con, cmd)
	cmd.AddCommand(command.ImplantCmd(con))
	carapace.Gen(cmd)

	return cmd, nil
}
