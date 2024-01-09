package cli

import (
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/console"
)

func StartConsole() error {
	err := console.Start(command.BindClientsCommands, command.BindImplantCommands)
	if err != nil {
		return err
	}
	return nil
}
