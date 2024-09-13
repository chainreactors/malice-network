package cli

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"os"
)

func StartConsole() error {
	var rootCmd = &cobra.Command{
		Use:   "client",
		Short: "",
		Long:  ``,
	}
	rootCmd.TraverseChildren = true

	con, err := repl.NewConsole()
	if err != nil {
		return err
	}

	rootCmd.AddCommand(command.ConsoleCmd(con))
	rootCmd.AddCommand(command.ImplantCmd(con))
	rootCmd.RunE, rootCmd.PostRunE = command.ConsoleRunnerCmd(con, true)
	carapace.Gen(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("root command: %s\n", err)
		os.Exit(1)
	}
	return nil
}
