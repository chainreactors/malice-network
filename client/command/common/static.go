package common

import (
	"fmt"
	"os"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var StdinIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func ShouldUseStaticOutput(con *core.Console) bool {
	if con != nil {
		return con.IsNonInteractiveExecution()
	}

	return !StdinIsTerminal()
}

func RunTable(con *core.Console, tableModel *tui.TableModel) (bool, error) {
	if ShouldUseStaticOutput(con) {
		if con != nil {
			con.Log.Console(tableModel.View())
		}
		return true, nil
	}

	return false, tableModel.Run()
}

func Confirm(cmd *cobra.Command, con *core.Console, prompt string) (bool, error) {
	if cmd != nil && cmd.Flags() != nil && cmd.Flags().Lookup("yes") != nil {
		yes, err := cmd.Flags().GetBool("yes")
		if err == nil && yes {
			return true, nil
		}
	}

	if ShouldUseStaticOutput(con) {
		return false, fmt.Errorf("command requires interactive confirmation; rerun with --yes or use an interactive terminal")
	}

	confirmModel := tui.NewConfirm(prompt)
	if err := confirmModel.Run(); err != nil {
		return false, err
	}

	return confirmModel.GetConfirmed(), nil
}
