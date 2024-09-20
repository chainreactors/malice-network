package cli

import (
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func rootCmd(con *repl.Console) (*cobra.Command, error) {
	var cmd = &cobra.Command{
		Use:   "client",
		Short: "",
		Long:  ``,
	}
	cmd.TraverseChildren = true

	cmd.AddCommand(command.ConsoleCmd(con))
	cmd.AddCommand(command.ImplantCmd(con))
	cmd.RunE, cmd.PostRunE = command.ConsoleRunnerCmd(con, true)
	carapace.Gen(cmd)

	return cmd, nil
}
