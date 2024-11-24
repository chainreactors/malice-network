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
	bind := command.MakeBind(cmd, con)
	command.BindCommonCommands(bind)
	cmd.PersistentPreRunE, cmd.PersistentPostRunE = command.ConsoleRunnerCmd(con, cmd)
	//cmd.AddCommand(command.ConsoleCmd(con))
	cmd.AddCommand(command.ImplantCmd(con))
	carapace.Gen(cmd)

	return cmd, nil
}
