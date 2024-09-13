package cli

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"os"
)

func StartConsole() error {
	con, err := repl.NewConsole()
	if err != nil {
		return err
	}
	cmd, err := rootCmd(con)
	if err != nil {
		return err
	}
	if err := cmd.Execute(); err != nil {
		fmt.Printf("root command: %s\n", err)
		os.Exit(1)
	}
	return nil
}
