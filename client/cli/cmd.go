package cli

import (
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/console"
)

func StartConsole() error {
	err := console.Start(command.BindCommands)
	if err != nil {
		return err
	}
	return nil
}
