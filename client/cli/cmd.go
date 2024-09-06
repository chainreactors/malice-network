package cli

import (
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/repl"
)

func StartConsole() error {
	err := repl.Start(command.BindClientsCommands, command.BindImplantCommands)
	if err != nil {
		return err
	}
	return nil
}
